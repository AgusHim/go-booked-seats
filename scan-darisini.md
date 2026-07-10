curl 'https://scanner.darisini.com/api/graphql' \
  -H 'accept: application/graphql-response+json; charset=utf-8, application/json; charset=utf-8' \
  -H 'accept-language: en-US,en;q=0.9' \
  -H 'content-type: application/json' \
  -b '__Host-next-auth.csrf-token=c8e11305d489a2f96356c8ba9ad6cc1cb80d51a503497c8b75e4f148307b5a76%7C8c90a3c369ea67984062e5d9967388179d21f0b449a612f3b22f703344fded75; __Secure-next-auth.callback-url=https%3A%2F%2Fscanner.darisini.com%2Fv2%2Flogin%3FscannerId%3Dcmrbwhfeb00x5s601wupxujif; __Secure-next-auth.session-token=eyJhbGciOiJkaXIiLCJlbmMiOiJBMjU2R0NNIn0..tOy6pQ-b8dLRByRe.mrjTq4bsE60ZSCYuMQ8HC3it4GkLYg7cKjxgmBgmSTMFGzs8X5YI-ubsSghieYjiCpSS0l2FqiObJ2pIMFO_uZmW3pZodFGpWGSbunsfflzXjPJfJr-0HOHA9u20pT8q5PoeoKNqyq9vb4rYk5JFbug6WQzrl-y9kZpOE4GJNBrVoRNlowOt0J0cHYiuzik0U7qsg5QYNqqx8mVV1FKnywsA5T-IyG9wEdQ8BfNJgIigyYG008IyZ2LUnb49bULS9TTSCmi6pe87wpyF_uj7NB8g5ZAfBcmNzwZl2N43sdtXWHsCF8VpuJFv3qWYqHkayV9NpYLJNwFZXwYgcJFClU1Gpd0db7RtRaNnyKBOy2nOT08V01Z8MgwP9a9Y6gM.boRW9LFjLwvKkkDx5NsobQ' \
  -H 'origin: https://scanner.darisini.com' \
  -H 'priority: u=1, i' \
  -H 'referer: https://scanner.darisini.com/v2/presence' \
  -H 'sec-ch-ua: "Google Chrome";v="149", "Chromium";v="149", "Not)A;Brand";v="24"' \
  -H 'sec-ch-ua-mobile: ?1' \
  -H 'sec-ch-ua-platform: "Android"' \
  -H 'sec-fetch-dest: empty' \
  -H 'sec-fetch-mode: cors' \
  -H 'sec-fetch-site: same-origin' \
  -H 'user-agent: Mozilla/5.0 (Linux; Android 15; Pixel 9) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Mobile Safari/537.36' \
  --data-raw $'{"query":"query useEventScannerValidateUserTicketQuery(\\n  $eventScannerId: ID\u0021\\n  $password: String\u0021\\n  $publicId: String\u0021\\n) {\\n  eventScannerValidateUserTicket(eventScannerId: $eventScannerId, password: $password, publicId: $publicId) {\\n    success\\n    error {\\n      code\\n      message\\n      ticketName\\n      eventTitle\\n      eventShortUrl\\n    }\\n    data {\\n      publicId\\n      orderUserEmail\\n      orderUserFullName\\n      ownerUserEmail\\n      ownerUserFullName\\n      ownerUserGender\\n      ticket {\\n        name\\n        eventTitle\\n        eventStartDate\\n      }\\n      attendance {\\n        decodedId\\n        attendedAt\\n        scannerUserFullName\\n        notes\\n        attachmentUrl\\n      }\\n      maximumScan\\n      currentScanCount\\n    }\\n  }\\n}\\n","variables":{"eventScannerId":null,"password":null,"publicId":"0ZHVCXMT"}}'


  Response:
  {
    "data": {
        "eventScannerValidateUserTicket": {
            "success": true,
            "error": null,
            "data": {
                "publicId": "0ZHVCXMT",
                "orderUserEmail": "himawan.ags@gmail.com",
                "orderUserFullName": "Agus Himawan",
                "ownerUserEmail": "silver4@gmail.com",
                "ownerUserFullName": "silver4@gmail.com",
                "ownerUserGender": "MALE",
                "ticket": {
                    "name": "Tiket Silver Tribun",
                    "eventTitle": "Just For Test",
                    "eventStartDate": "2039-06-16T17:00:00.000Z"
                },
                "attendance": null,
                "maximumScan": 1,
                "currentScanCount": 0
            }
        }
    }
}