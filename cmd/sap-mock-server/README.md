# sap-mock-server

Servidor HTTP que simula un servicio OData v2 de SAP NetWeaver para desarrollo
local e integración con el microservicio `ms-neg-standard-pe-odata-demo`.

## Características

- Responde a todos los endpoints OData usados por el microservicio
- Imprime en consola **todos los detalles** de cada request:
  - Método, URL completa, protocolo y dirección remota
  - Credenciales Basic Auth (contraseña enmascarada)
  - Todas las cabeceras HTTP
  - Query params OData (`$top`, `$skip`, `$filter`, `sap-language`, etc.)
  - Análisis del path OData: entity set y key predicate
  - Body completo con pretty-print JSON
- Simula el flujo completo de **CSRF token** (fetch → validación → uso único)
- Datos de prueba incluidos (productos, company codes, sales orders)
- Sin dependencias externas — solo stdlib de Go

## Compilar y ejecutar

```bash
# desde la raíz del repo traverse:
cd cmd/sap-mock-server
GOWORK=off go run .

# o compilar un binario:
GOWORK=off go build -o sap-mock-server .
./sap-mock-server
```

## Flags

| Flag       | Default     | Descripción                                      |
|------------|-------------|--------------------------------------------------|
| `-addr`    | `:44300`    | Dirección y puerto donde escucha el servidor     |
| `-user`    | `sapuser`   | Usuario esperado en Basic Auth                   |
| `-pass`    | `sappass`   | Contraseña esperada en Basic Auth                |
| `-noauth`  | `false`     | Deshabilitar validación de Basic Auth            |
| `-nocolor` | `false`     | Deshabilitar colores ANSI en la salida           |
| `-delay`   | `0`         | Retardo artificial en milisegundos por request   |

## Endpoints simulados

```
GET  /sap/opu/odata/sap/UI_PRODUCTLIST/$metadata
GET  /sap/opu/odata/sap/UI_PRODUCTLIST/ProductList
GET  /sap/opu/odata/sap/UI_PRODUCTLIST/ProductList(Product='X',Plant='Y',ValuationType='Z')

GET  /sap/opu/odata/sap/API_COMPANYCODE_SRV/$metadata
GET  /sap/opu/odata/sap/API_COMPANYCODE_SRV/A_CompanyCode
GET  /sap/opu/odata/sap/API_COMPANYCODE_SRV/A_CompanyCode('1000')

GET  /sap/opu/odata/sap/SD_SALES_ORDER_IMPORT/$metadata
GET  /sap/opu/odata/sap/SD_SALES_ORDER_IMPORT/I_SalesOrderImport

GET|POST /sap/opu/odata/sap/API_MAINTNOTIFICATION/$metadata    [X-CSRF-Token: Fetch]
POST     /sap/opu/odata/sap/API_MAINTNOTIFICATION/MaintenanceNotification  [CSRF required]

GET /health
```

## Flujo CSRF (Notifications)

```
1. POST $metadata con cabecera  X-CSRF-Token: Fetch
   ← respuesta incluye  X-CSRF-Token: <token>

2. POST MaintenanceNotification con cabecera  X-CSRF-Token: <token>
   ← 201 Created con el número de notificación generado

   Si el token es inválido, expirado o ya fue usado → 403 Forbidden
```

## Configurar el microservicio para usar el mock

En `ms-neg-standard-pe-odata-demo/.env`:

```env
SAP_ODATA_HOST=http://localhost:44300
SAP_ODATA_USER=sapuser
SAP_ODATA_PASS=sappass
SAP_INSECURE_TLS=false
```
