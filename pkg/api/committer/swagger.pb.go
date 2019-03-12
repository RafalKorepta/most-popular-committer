package committer

const (
	swagger = `{
  "swagger": "2.0",
  "info": {
    "title": "committer.proto",
    "version": "version not set"
  },
  "schemes": [
    "http",
    "https"
  ],
  "consumes": [
    "application/json"
  ],
  "produces": [
    "application/json"
  ],
  "paths": {
    "/v1alpha1/committer": {
      "get": {
        "summary": "SendMail",
        "operationId": "MostActiveCommitter",
        "responses": {
          "200": {
            "description": "A successful response.",
            "schema": {
              "$ref": "#/definitions/v1alpha1CommitterResponse"
            }
          }
        },
        "parameters": [
          {
            "name": "language",
            "in": "query",
            "required": false,
            "type": "string"
          }
        ],
        "tags": [
          "CommitterService"
        ]
      }
    }
  },
  "definitions": {
    "v1alpha1Committer": {
      "type": "object",
      "properties": {
        "name": {
          "type": "string"
        },
        "commits": {
          "type": "string",
          "format": "uint64"
        }
      }
    },
    "v1alpha1CommitterResponse": {
      "type": "object",
      "properties": {
        "language": {
          "type": "string"
        },
        "contributors": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/v1alpha1Committer"
          }
        }
      }
    }
  }
}
`
)
