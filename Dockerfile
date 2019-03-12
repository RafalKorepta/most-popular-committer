FROM scratch

COPY dist/most-popular-committer most-popular-committer

EXPOSE 9091

CMD ["/most-popular-committer", "serve"]