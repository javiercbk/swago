# Swago

Autogenerate the `swagger` documentation for your services

[Golangci report](https://golangci.com/r/github.com/javiercbk/swago)

## Wait what?

By reading the `ast` of your source files, `swago` attempts to extract the routes, the handlers of each route, the request and response model

## How does it perform such magic

Well, it is not THAT magic, but it is quite cool. By reading a config file, it will search for certain patterns in order to extract routes, handlers, request and response models.

It will search in every function of the project for patterns and autogenerate the `swagger.yml` file for you. As a developer, you only need to follow the same patterns over and over again, which is quite common in GO.


