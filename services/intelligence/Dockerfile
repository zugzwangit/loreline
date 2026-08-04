FROM python:3.12.13-slim AS builder
WORKDIR /build
COPY pyproject.toml ./
COPY loreline_ai ./loreline_ai
RUN pip wheel --no-cache-dir --no-deps --wheel-dir /wheels .

FROM python:3.12.13-slim
WORKDIR /app
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1
COPY requirements.lock ./
COPY --from=builder /wheels /wheels
RUN pip install --no-cache-dir --require-hashes -r requirements.lock \
 && pip install --no-cache-dir --no-deps /wheels/*.whl \
 && rm -rf /wheels
USER 65532:65532
EXPOSE 8090
CMD ["loreline-intelligence"]
