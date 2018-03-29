# Use an official Python runtime as a parent image
FROM python:3.6-slim

# Install memcached
RUN apt-get update -qq && \
    apt-get install -yqq --no-install-recommends memcached && \
    apt-get -yqq clean && \
    apt-get -yqq purge && \
    rm -rf /var/lib/apt/lists/*

# Set the working directory to /app
WORKDIR /app

# Copy the requirements.txt into the container at /app
ADD requirements.txt /app

# Install any needed packages specified in requirements.txt
RUN pip install --trusted-host pypi.python.org -r requirements.txt

# Copy the current directory contents into the container at /app
ADD . /app

# Make port 8000 available to the world outside this container
EXPOSE 8000

# Run app with memcached when the container launches
CMD ["sh", "-c", "memcached -d -u memcache -m 128 && exec gunicorn -w 4 -b :8000 -k gevent fullrss:app"]