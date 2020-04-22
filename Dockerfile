FROM node:14-buster as builder

RUN npm --version
RUN npm update -g && npm install -g --save redoc@v2.0.0-rc.27 redoc-cli speccy \
        dompurify@2.0.8 # see https://github.com/Redocly/redoc/issues/1242

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
