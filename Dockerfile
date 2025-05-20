# Dockerfile for local testing with PHP-FPM
FROM php:8.4-fpm-bookworm

# Install extensions and tools
RUN apt-get update && \
    apt-get install -y bash curl git build-essential wget && \
    docker-php-ext-install opcache

# Install Go
ENV GO_VERSION=1.22.2
RUN wget https://go.dev/dl/go$GO_VERSION.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go$GO_VERSION.linux-amd64.tar.gz && \
    rm go$GO_VERSION.linux-amd64.tar.gz
ENV PATH=$PATH:/usr/local/go/bin

# Configure OPcache
RUN echo "opcache.enable=1" >> /usr/local/etc/php/conf.d/opcache.ini && \
    echo "opcache.enable_cli=1" >> /usr/local/etc/php/conf.d/opcache.ini && \
    echo "opcache.memory_consumption=128" >> /usr/local/etc/php/conf.d/opcache.ini && \
    echo "opcache.interned_strings_buffer=8" >> /usr/local/etc/php/conf.d/opcache.ini && \
    echo "opcache.max_accelerated_files=10000" >> /usr/local/etc/php/conf.d/opcache.ini && \
    echo "opcache.validate_timestamps=1" >> /usr/local/etc/php/conf.d/opcache.ini && \
    echo "opcache.revalidate_freq=2" >> /usr/local/etc/php/conf.d/opcache.ini

# Add simple status endpoint
RUN echo "<?php phpinfo(); ?>" > /var/www/html/status.php

# Set working directory
WORKDIR /var/www/html

# Expose FPM socket (optional)
EXPOSE 9000

CMD ["bash"]