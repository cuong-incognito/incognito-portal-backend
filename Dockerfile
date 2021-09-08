FROM golang:1.17-buster AS build

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -tags=jsoniter -ldflags "-linkmode external -extldflags -static" -o portal-backend


FROM alpine

WORKDIR /app

COPY --from=build /app/portal-backend /app/portal-backend

CMD [ "./portal-backend" ]
