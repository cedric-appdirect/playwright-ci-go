FROM mcr.microsoft.com/playwright:v1.48.2 AS pw-server

WORKDIR /src
ENV PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1

RUN npm init -y
RUN npm install node
RUN npm install @playwright/test@1.48.2
RUN npx -y playwright@1.48.2 install --with-deps

COPY remote-playwright.js remote-playwright.js

ENTRYPOINT [ "/bin/sh", "-c" ]