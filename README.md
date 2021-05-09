# Sandissa: Server

```
           ███████  █████  ███    ██ ██████  ██ ███████ ███████  █████
           ██      ██   ██ ████   ██ ██   ██ ██ ██      ██      ██   ██
           ███████ ███████ ██ ██  ██ ██   ██ ██ ███████ ███████ ███████
                ██ ██   ██ ██  ██ ██ ██   ██ ██      ██      ██ ██   ██
           ███████ ██   ██ ██   ████ ██████  ██ ███████ ███████ ██   ██

                                Server for Sandissa
                 Handle incoming REST, verify auth, and call MQTT
```

This repository is a part of the final EECS 700: IoT Security project. This
repository holds code for the server that served as an intermediary between
IoT smart devices and users. It also provided auth services and data storing.

Please note that all the keys and certificates in this repository are invalid,
revoked, and inactive. Any attempt on utilizing them in any environment would
not lead to desired effect. Those certificates were built on a local isolated 
machine that itself served as CA. The infrastructure has been dismantled and
therefore it's all invalid as of now. Feel free to inspect it. 

For more information on the project, visit the 
[root repository of the project](https://github.com/thecsw/sandissa-dev).
