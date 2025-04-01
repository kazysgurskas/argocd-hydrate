FROM --platform=$TARGETPLATFORM debian:stable-slim

COPY argocd-hydrate /argocd-hydrate

ENTRYPOINT ["/argocd-hydrate"]
