FROM node:10-jessie as builder
RUN npm update -g && npm install -g --save call-me-maybe mobx styled-components react react-dom \
        redoc redoc-cli speccy core-js rxjs typescript mobx \
        base64-js ieee754 isarray inherits readable-stream to-arraybuffer xtend builtin-status-codes

COPY openapi.yaml openapi.yaml

RUN speccy lint openapi.yaml
RUN redoc-cli bundle openapi.yaml

FROM nginx:1.16
RUN chmod 777 -R /var/cache/nginx/
COPY nginx/nginx.conf /opt/nginx/nginx.conf
COPY nginx/default.conf /opt/nginx/conf.d/default.conf

COPY --from=builder redoc-static.html /opt/nginx/www/index.html
RUN echo '# empty prometheus metrics response' > /opt/nginx/www/metrics

RUN adduser -q --gecos '' --disabled-login --shell /bin/false moov
USER moov

EXPOSE 8080
ENTRYPOINT ["nginx"]
CMD ["-c", "/opt/nginx/nginx.conf"]
