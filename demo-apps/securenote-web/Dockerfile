FROM node:20-slim AS builder

WORKDIR /app

# Build argument for API URL (can be overridden in docker-compose)
# API URL のビルド引数（docker-composeで上書き可能）
ARG VITE_API_URL=http://api.securenote.test:8000/api
ENV VITE_API_URL=$VITE_API_URL

# Copy package files
COPY package*.json ./

# Install dependencies
# Use npm install to auto-generate package-lock.json on first run
RUN npm install

# Copy source code
COPY . .

# Build with environment variable
RUN npm run build

# Production image
FROM nginx:alpine

# Copy built files
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy nginx config
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
