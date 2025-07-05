You can deploy this project on Cloud Run by navigating to the `google_adk`
folder, and running the following command:

```bash
gcloud run deploy my-adk-project --source . --base-image python312 --set-build-env-vars GOOGLE_ENTRYPOINT="adk web --host 0.0.0.0 --port 8080"
```