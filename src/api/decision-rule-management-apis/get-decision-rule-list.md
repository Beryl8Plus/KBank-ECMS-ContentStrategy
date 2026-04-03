List view Decision Rule:
GET : /decision-rule/list?type=string&keyword=string&status=string&page=number&size=number

## Response:

```json
{
    "data": [
        {
            "id": number,
            "decisionName": string,
            "type": string,
            "placements": string,
            "status": string,
            "created_at": string,
            "updated_at": string
        }
    ],
    "pagination": {
        "total": number,
        "page": number,
        "size": number
    }
}
```
