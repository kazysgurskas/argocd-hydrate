FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o argocd-hydrate ./cmd/argocd-hydrate


FROM gcr.io/distroless/static-debian11:nonroot

COPY --from=builder /app/argocd-hydrate /argocd-hydrate

WORKDIR /workspace

ENTRYPOINT ["/argocd-hydrate"]
