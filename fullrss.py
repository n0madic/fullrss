from multiprocessing.dummy import Pool as ThreadPool
from flask import Flask
from flask import abort
from flask import jsonify
from flask import request
from flask import redirect
from flask import render_template
from flask import send_from_directory

app = Flask(__name__)

from datetime import datetime
from lxml.html.clean import clean_html
from pyquery import PyQuery
from readability import Document
from urllib.parse import urljoin
from werkzeug.contrib.atom import AtomFeed
from werkzeug.contrib.cache import MemcachedCache
import feedparser
import lxml.html
import os
import requests
import time
import yaml


class FullFeed(object):
    def __init__(self, url, base_href="", description='', method='readability', method_query='', filters=None):
        self.description = description
        self.url = url
        self.base_href = base_href if base_href else urljoin(url, '/')
        self.method = method
        self.method_query = method_query
        self.filters = filters if filters else {}

    def full_summary(self, entry):
        try:
            html = config.memcache.get(entry.link)
            if not html:
                headers = {
                    'User-Agent': 'FullRSS proxy'
                }
                try:
                    response = requests.get(entry.link, headers=headers, timeout=(3, 10))
                    if response.status_code == 200 and response.text:
                        if response.encoding == 'ISO-8859-1':
                            response.encoding = response.apparent_encoding
                        config.memcache.set(entry.link, response.text)
                        html = response.text
                    else:
                        response.raise_for_status()
                except requests.exceptions.RequestException as e:
                    app.logger.error('Download url {} failed:\n{}'.format(entry.link, e))
            if self.method == 'readability':
                doc = Document(html)
                lx = lxml.html.fromstring(doc.summary(html_partial=True))
            elif self.method == 'pyquery':
                doc = PyQuery(html)
                lx = lxml.html.fromstring(doc(self.method_query).outerHtml())
            elif self.method == 'xpath':
                doc = lxml.html.fromstring(html)
                lx = doc.xpath(self.method_query)[0]
            if 'selectors' in self.filters:
                for selector in self.filters['selectors']:
                    for bad in lx.cssselect(selector):
                        bad.getparent().remove(bad)
            if 'text' in self.filters:
                for text in self.filters['text']:
                    for bad in lx.xpath('.//*[contains(text(),"{}")]'.format(text)):
                        bad.getparent().remove(bad)
            lx.make_links_absolute(self.base_href)
            entry.full_summary = lxml.html.tostring(clean_html(lx), encoding='unicode')
        except Exception as e:
            app.logger.error('Problem with URL {}:\n{}'.format(entry.link, e))
            entry.full_summary = entry.summary
        return entry

    def get(self):
        mimetypes = {'text/html': 'html', 'text/plain': 'text'}
        parsed = feedparser.parse(self.url)
        if parsed.bozo:
            app.logger.error(parsed.bozo_exception)
        if parsed.entries:
            feed_updated = parsed.feed.get('updated_parsed', parsed.feed.get('published_parsed', time.localtime()))
            feed = AtomFeed(title=parsed.feed.get('title', self.description),
                            title_type=mimetypes[parsed.feed.title_detail.type],
                            subtitle=parsed.feed.subtitle,
                            subtitle_type=mimetypes[parsed.feed.subtitle_detail.type],
                            author=parsed.feed.author if 'author' in parsed.feed else None,
                            feed_url=self.url,
                            url=parsed.feed.link,
                            logo=parsed.feed.get('logo'),
                            icon=parsed.feed.get('icon'),
                            links=parsed.feed.get('links', []),
                            generator=('fullrss.py by Nomadic',
                                       'https://github.com/n0madic/fullrss', '0.1')
                            )
            for entry in ThreadPool(10).map(self.full_summary, parsed.entries):
                try:
                    feed.add(title=entry.title,
                             title_type=mimetypes[entry.title_detail.type],
                             summary=entry.full_summary,
                             summary_type='html',
                             author=entry.get('author'),
                             url=entry.link,
                             links=entry.get('links', []),
                             id=entry.get('guid', entry.link),
                             updated=datetime(*entry.get('updated_parsed', feed_updated)[:6]),
                             published=datetime(*entry.get('published_parsed', feed_updated)[:6]),
                             categories=entry.tags if 'tags' in entry else []
                             )
                except ValueError as e:
                    app.logger.error('Skipped feed entry {} ({}):\n{}'.format(entry.title, entry.link, e))
            return feed.get_response()
        else:
            return repr(parsed.bozo_exception), 503


class Config(dict):
    def __init__(self, filename):
        dict.__init__(self)
        self.filename = filename
        self.feeds = {}
        self.load()

    def load(self):
        with open(self.filename, encoding='utf-8') as config_file:
            self.update(yaml.load(config_file))
        self.set_feeds()
        self.set_memcache()

    def set_feeds(self):
        for feed, kargs in self['feeds'].items():
            self.set_feed(feed, **kargs)

    def set_feed(self, name, **kargs):
        self.feeds[name] = FullFeed(**kargs)

    def set_memcache(self):
        memcached_host = self['settings'].get('memcache', '127.0.0.1:11211')
        self.memcache = MemcachedCache([memcached_host])


config = Config('fullrss.yaml')


@app.before_first_request
def startup():
    pass


@app.route('/')
@app.route('/index.html')
def index():
    feed = request.args.get('feed', type=str)
    if feed:
        return redirect("/feed/" + feed, code=301)
    return render_template("index.html", feeds=config['feeds'])


@app.route('/favicon.ico')
def favicon():
    return send_from_directory(os.path.join(app.root_path, 'static'),
                               'favicon.ico', mimetype='image/vnd.microsoft.icon')


@app.route('/feed/<feedname>')
def get_feed(feedname):
    if feedname in config.feeds:
        return config.feeds[feedname].get()
    else:
        abort(404)


@app.route('/config/<output_format>')
def get_config(output_format):
    if output_format == 'yaml':
        return yaml.dump(dict(config), allow_unicode=True, default_flow_style=False), {
            'Content-Type': 'text/yaml; charset=utf-8'}
    if output_format == 'json':
        return jsonify(config)


if __name__ == "__main__":
    from werkzeug.contrib.profiler import ProfilerMiddleware

    app.config['JSON_AS_ASCII'] = False
    app.config['PROFILE'] = False
    if app.config['PROFILE']:
        app.wsgi_app = ProfilerMiddleware(app.wsgi_app, restrictions=[30])
    app.run(threaded=True, debug=True)
