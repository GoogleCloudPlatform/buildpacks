You can deploy this project on Cloud Run by navigating to the `google_adk`
folder, running either of the following commands:

1. With UI option

```bash
gcloud run deploy my-adk-project --source . --base-image python312 --set-build-env-vars GOOGLE_ENTRYPOINT="adk web --host 0.0.0.0 --port 8080"
```

2. api_server option

```bash
gcloud run deploy my-adk-project --source . --base-image python312 --set-build-env-vars GOOGLE_ENTRYPOINT="adk api_server --host 0.0.0.0 --port 8080"
```