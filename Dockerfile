FROM scratch
COPY skate /usr/local/bin/skate

# Set the default command
ENTRYPOINT [ "/usr/local/bin/skate" ]
