FROM node:20-slim AS builder
WORKDIR /app
COPY package*.json ./
RUN npm install

FROM node:20-slim
WORKDIR /app
COPY --from=builder /app/node_modules ./node_modules
COPY package*.json ./
COPY src/ ./src/

# Create secrets directory (will be mounted or populated at runtime)
RUN mkdir -p /app/secrets

# Run as non-root user
USER node

EXPOSE 8080

CMD ["npm", "start"]
