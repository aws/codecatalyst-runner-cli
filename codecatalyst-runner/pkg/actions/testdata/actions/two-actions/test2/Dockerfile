ARG AL_VERSION=2022

FROM public.ecr.aws/amazonlinux/amazonlinux:${AL_VERSION}

WORKDIR /app
COPY main.sh .

ENTRYPOINT ["/app/main.sh"]
