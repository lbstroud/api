FROM node:8-jessie as builder
RUN npm update -g && npm install -g --save mobx styled-components react react-dom redoc redoc-cli@0.7.0 speccy core-js rxjs typescript mobx

COPY openapi.yaml openapi.yaml

RUN speccy lint openapi.yaml
RUN redoc-cli bundle openapi.yaml

FROM nginx:1.15
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
