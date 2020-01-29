FROM node:13-buster as builder
RUN npm update -g && npm install -g --save redoc redoc-cli speccy

COPY openapi.yaml openapi.yaml

RUN speccy lint openapi.yaml
RUN redoc-cli bundle openapi.yaml \
        --options.theme.menu.backgroundColor="#263238" \
        --options.theme.menu.textColor="#ffffff" \
        --options.theme.menu.rightPanel.backgroundColor="#263238" \
        --options.theme.menu.rightPanel.textColor="#333333" \
        --options.nativeScrollbars

FROM nginx:1.17
USER nginx

COPY nginx/nginx.conf /opt/nginx/nginx.conf
COPY nginx/default.conf /opt/nginx/conf.d/default.conf
COPY nginx/metrics /opt/nginx/www/metrics

COPY ./site/ /opt/nginx/www/

COPY --from=builder redoc-static.html /opt/nginx/www/v1/index.html

EXPOSE 8080
ENTRYPOINT ["nginx"]
CMD ["-c", "/opt/nginx/nginx.conf"]
