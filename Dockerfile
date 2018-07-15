FROM node:alpine

RUN mkdir /app

COPY ./package.json /app

WORKDIR /app

RUN npm install

COPY . /app

CMD [ "node", "/app/poller.js" ]