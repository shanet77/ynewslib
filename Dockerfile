FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /ynewslib .

FROM scratch
COPY --from=build /ynewslib /ynewslib
EXPOSE 8080
ENTRYPOINT ["/ynewslib"]
