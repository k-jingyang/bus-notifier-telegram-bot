#!/bin/bash
LTA_API_TOKEN=$(grep LTA_API_TOKEN ../.env | cut -d '=' -f 2-)

curl -H "AccountKey: $LTA_API_TOKEN" 'http://datamall2.mytransport.sg/ltaodataservice/BusServices' \
| jq -r .value[].ServiceNo \
| uniq > bus_services.txt


curl -H "AccountKey: $LTA_API_TOKEN" 'http://datamall2.mytransport.sg/ltaodataservice/BusServices?$skip=500' \
| jq -r .value[].ServiceNo \
| uniq >> bus_services.txt

cd refdatadownloader && go run reference_data_downloader.go
