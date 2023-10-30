FROM python:3.11-slim-bookworm as envbuilder

ARG POETRY_VERSION=1.6

ENV POETRY_NO_INTERACTION=1 \
    POETRY_VIRTUALENVS_IN_PROJECT=1 \
    POETRY_VIRTUALENVS_CREATE=1

RUN apt-get update \
    && apt-get install --no-install-recommends -y build-essential \
    && apt-get clean && rm -rf /var/lib/apt/lists/* \
    && pip install --no-cache-dir poetry==$POETRY_VERSION

WORKDIR /app
COPY poetry.lock pyproject.toml ./

RUN poetry install --without dev --no-root --no-cache \
    && pip cache purge \
    && apt-get purge -y --auto-remove build-essential


FROM python:3.11-slim-bookworm AS runtime

RUN apt-get update \
    && apt-get install --no-install-recommends -y dumb-init \
    && apt-get clean && rm -rf /var/lib/apt/lists/* \
    && apt-get purge -y --auto-remove

ENV VIRTUAL_ENV=/app/.venv
COPY --from=envbuilder ${VIRTUAL_ENV} ${VIRTUAL_ENV}
ENV PATH="$VIRTUAL_ENV/bin:$PATH"

COPY . /app
WORKDIR /app

ENTRYPOINT ["/usr/bin/dumb-init", "--"]
EXPOSE 5000