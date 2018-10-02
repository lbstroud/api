FROM node:8-jessie as builder
RUN npm install -g --save mobx styled-components react react-dom redoc redoc-cli

COPY openapi.yaml openapi.yaml
RUN redoc-cli bundle openapi.yaml

FROM nginx:1.15
RUN chmod 777 -R /var/cache/nginx/
COPY nginx/nginx.conf /opt/nginx/nginx.conf
COPY nginx/default.conf /opt/nginx/conf.d/default.conf

COPY --from=builder redoc-static.html /opt/nginx/www/index.html

USER nobody
EXPOSE 8080
ENTRYPOINT ["nginx"]
CMD ["-c", "/opt/nginx/nginx.conf"]
