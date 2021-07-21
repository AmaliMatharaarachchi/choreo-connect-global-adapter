from mcr.microsoft.com/mssql/server:2019-latest

ENV ACCEPT_EULA=Y
ENV SA_PASSWORD=Admin001

COPY maven/entrypoint.sh .
COPY maven/init-db.sh .
COPY maven/init.sql .

USER mssql
ENTRYPOINT /bin/bash ./entrypoint.sh
