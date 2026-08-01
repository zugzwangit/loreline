FROM python:3.12-slim
WORKDIR /app
COPY pyproject.toml ./
COPY loreline_ai ./loreline_ai
RUN pip install --no-cache-dir .
USER 65532:65532
EXPOSE 8090
CMD ["loreline-intelligence"]
