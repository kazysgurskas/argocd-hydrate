FROM --platform=$TARGETPLATFORM debian:stable-slim

COPY argocd-diff /argocd-diff

ENTRYPOINT ["/argocd-diff"]
