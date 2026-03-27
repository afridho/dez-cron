package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const openAPISpec = `
openapi: 3.0.0
info:
  title: Dez Cron API
  version: 1.0.0
  description: "A fast, MongoDB integrated API for creating and managing dynamic HTTP Cron Jobs."
servers:
  - url: /api/jobs
    description: Base URL
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: API Token
security:
  - bearerAuth: []
paths:
  /:
    get:
      summary: List all Cron Jobs
      responses:
        '200':
          description: A JSON array of Cron jobs
    post:
      summary: Create a new Cron Job
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                title:
                  type: string
                  example: "Ping Backup API"
                url:
                  type: string
                  example: "https://your-api.com/backup"
                method:
                  type: string
                  example: "POST"
                schedule:
                  type: string
                  example: "0 0 * * *"
                timezone:
                  type: string
                  example: "Asia/Jakarta"
                is_active:
                  type: boolean
                  example: true
                retry_count:
                  type: integer
                  example: 5
                disabled_after:
                  type: integer
                  example: 20
                headers:
                  type: object
                  example: {"Authorization": "Bearer secret"}
                body:
                  type: string
                  example: '{"backup":true}'
      responses:
        '201':
          description: Created
  /{id}:
    get:
      summary: Get a single Cron Job by ID
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Success
    put:
      summary: Update an existing Cron Job
      parameters:
        - name: id
          in: path
          required: true
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
      responses:
        '200':
          description: Updated successfully
    delete:
      summary: Delete a Cron Job
      parameters:
        - name: id
          in: path
          required: true
      responses:
        '200':
          description: Deleted successfully
  /logs:
    get:
      summary: List all Cron Job execution logs
      responses:
        '200':
          description: A list of logs
  /logs/{job_id}:
    get:
      summary: Get execution logs for a specific Job ID
      parameters:
        - name: job_id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Logs array
`

func ShowDocs(c *gin.Context) {
	html := `<!DOCTYPE html>
<html>
  <head>
    <title>Dez Cron API Reference</title>
    <!-- needed for adaptive design -->
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>body { margin: 0; padding: 0; }</style>
  </head>
  <body>
    <redoc spec-url='/api-docs.yaml'></redoc>
    <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"> </script>
  </body>
</html>`
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func ServeSpec(c *gin.Context) {
	c.Data(http.StatusOK, "application/yaml; charset=utf-8", []byte(openAPISpec))
}
