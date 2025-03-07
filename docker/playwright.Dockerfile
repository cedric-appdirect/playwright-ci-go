ARG PLAYWRIGHT_VERSION

FROM mcr.microsoft.com/playwright:v${PLAYWRIGHT_VERSION} AS pw-server

ARG PLAYWRIGHT_VERSION

WORKDIR /src
ENV PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1

RUN npm init -y
RUN npm install node
RUN npm install --save --save-exact @playwright/test@${PLAYWRIGHT_VERSION} && npm ls
RUN npx -y playwright@${PLAYWRIGHT_VERSION} install --with-deps

COPY remote-playwright.js remote-playwright.js

ENTRYPOINT [ "/bin/sh", "-c" ]