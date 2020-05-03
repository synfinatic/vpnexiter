from ubuntu:18.04
RUN mkdir /scripts
COPY ./scripts/* /scripts/

RUN /scripts/install_speedtest.sh

EXPOSE 5000/tcp 




