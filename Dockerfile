FROM gcr.io/distroless/static-debian11:nonroot

COPY argocd-hydrate /argocd-hydrate

ENTRYPOINT ["/argocd-hydrate"]
