FROM busybox
COPY output/bin/baetyl-cloud /bin/
COPY output/bin/baetyl-cloud.test /bin/
COPY output/templates /etc/templates
ENTRYPOINT [ "baetyl-cloud" ]
