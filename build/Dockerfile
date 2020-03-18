FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/application-monitoring-operator \
    USER_UID=1001 \
    USER_NAME=application-monitoring-operator 

# install operator binary
COPY build/_output/bin/application-monitoring-operator ${OPERATOR}
# copy templates
COPY templates /usr/local/bin/templates

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
