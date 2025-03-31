FROM golang:1.23-alpine AS builder

ARG VERSION=dev
ARG GIT_COMMIT=none
ARG BUILD_DATE=unknown

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags "-X github.com/kazysgurskas/argocd-hydrate/internal/cmd.getVersion.Version=${VERSION} \
              -X github.com/kazysgurskas/argocd-hydrate/internal/cmd.getVersion.GitCommit=${GIT_COMMIT} \
              -X github.com/kazysgurskas/argocd-hydrate/internal/cmd.getVersion.BuildDate=${BUILD_DATE}" \
    -o argocd-hydrate ./cmd/argocd-hydrate


FROM gcr.io/distroless/static-debian11:nonroot

COPY --from=builder /app/argocd-hydrate /argocd-hydrate

WORKDIR /workspace

ENTRYPOINT ["/argocd-hydrate"]
