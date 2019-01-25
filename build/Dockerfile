FROM alpine:3.8

RUN apk upgrade --update --no-cache

USER nobody

ADD templates/*.yaml /usr/local/bin/templates/
ADD build/_output/bin/application-monitoring-operator /usr/local/bin/application-monitoring-operator
