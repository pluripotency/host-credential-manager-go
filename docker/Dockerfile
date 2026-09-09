# Stage 1: Frontend Builder
FROM node:20-alpine AS frontend-builder
WORKDIR /app/front
COPY front/package*.json ./
RUN npm install
COPY front/ ./
RUN npm run build

# Stage 2: Go Backend Builder
FROM golang:1.26-alpine AS backend-builder
RUN apk add --no-cache bash
WORKDIR /app
ENV CGO_ENABLED=0 GOOS=linux
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
COPY go_src/ ./go_src/
COPY cert/ ./cert/
COPY hcm-client/ ./hcm-client/
COPY --from=frontend-builder /app/front/dist/ ./front/dist/
RUN go build -o host-credential-manager-go main.go
RUN ./hcm-client/build.sh

# Stage 3: Final Runtime
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=backend-builder /app/host-credential-manager-go .
COPY --from=backend-builder /app/hcm-client/built ./hcm-client/built
EXPOSE 8080
CMD ["./host-credential-manager-go"]
