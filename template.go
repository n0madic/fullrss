package main

const indexTpl = `<!DOCTYPE html>
<html>
<head>
    <title>Full text RSS feeds proxy</title>
    <meta http-equiv="Content-Type" content="text/html; charset=utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
    <meta name="keywords" content="RSS, Atom, feed, full, full text, full content, full article">
    <link href="favicon.ico" rel="icon" type="image/x-icon"/>
    <link rel="stylesheet" href="//maxcdn.bootstrapcdn.com/bootstrap/latest/css/bootstrap.min.css">
    <link rel="stylesheet" href="//maxcdn.bootstrapcdn.com/bootstrap/latest/css/bootstrap-theme.min.css">
</head>
<body>
<div class="container">
    <div class="jumbotron">
        <h2>Full text RSS feeds proxy
            <small>by Nomadic</small>
        </h2>
    </div>
    <div class="card bg-light">
        <h5 class="card-header"> Available feeds <img src="favicon.ico"></h5>
        <div class="card-body">
            <ol class="card-text">
				{{range $key, $value := .Feeds}}
                <li><a href="/feed/{{ $key }}" target="_blank">{{ $value.Description }}</a></li>
                {{end}}
            </ol>
        </div> <!-- card-body -->
    </div> <!-- card -->
    <footer>
        <div style="text-align: center;"><p><a href="https://github.com/n0madic/fullrss">GitHub</a> &copy; Nomadic 2018-2021
        </p></div>
    </footer>
</div> <!-- container -->
</body>
</html>`
