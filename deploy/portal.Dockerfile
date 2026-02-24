FROM node:22.17 AS build

WORKDIR /app

# Copy package files first for better caching
COPY portal/package.json portal/package-lock.json* ./

# Install dependencies
RUN npm ci

# Copy source code
COPY portal/ .

# Build the Angular application for production
RUN npm run build

FROM nginx:alpine
# Angular 17+ outputs to dist/portal/browser
# Remove default nginx content and copy our app to root
RUN rm -rf /usr/share/nginx/html/*
COPY --from=build /app/dist/portal/browser /usr/share/nginx/html
COPY deploy/nginx.conf /etc/nginx/nginx.conf
EXPOSE 8080