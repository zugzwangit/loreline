FROM python:3.12.13-slim
WORKDIR /app
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1
COPY requirements.lock ./
COPY pyproject.toml ./
COPY loreline_ai ./loreline_ai
RUN pip install --no-cache-dir --require-hashes -r requirements.lock \
 && pip install --no-cache-dir --no-deps --no-build-isolation .
USER 65532:65532
EXPOSE 8090
CMD ["loreline-intelligence"]
