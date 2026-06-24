FROM node:22-slim

WORKDIR /app

COPY package.json ./
RUN npm install --omit=dev

COPY server.js ./
COPY lib/ lib/
COPY public/ public/
COPY dist/ dist/

ENV PORT=10067
EXPOSE 10067
CMD ["node", "server.js"]
