# mapr_go_client_mqtt
A simple Go Client that reads from MQTT and writes to a MapR DB JSON database

To build a docker container:
docker build -t mapr_go_client_mqtt .

to run the docker container:
docker run mapr_go_client_mqtt which will show the parameters

To run with parameters:
docker run --rm -it mapr_go_client_mqtt -password mapr -mapr-url 10.0.0.11:5678 -mqtt-url 10.0.0.11:1883
