FROM node:20-bookworm

WORKDIR /app

COPY examples/hackathon-starter/frontend/package*.json ./
RUN npm ci

COPY examples/hackathon-starter/frontend ./

EXPOSE 5173

CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0", "--port", "5173", "--strictPort"]
