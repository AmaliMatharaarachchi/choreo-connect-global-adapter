#!/bin/bash
/opt/mssql/bin/sqlservr & ./init-db.sh & tail -f /dev/null
