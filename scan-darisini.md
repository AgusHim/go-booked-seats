 validate ticket dengan enpoint dibawah ini

## Confirm validate ticket
curl 'https://scanner.darisini.com/api/graphql' \
-X 'POST' \
-H 'Content-Type: application/json' \
-H 'Accept: application/graphql-response+json; charset=utf-8, application/json; charset=utf-8' \
-H 'Sec-Fetch-Site: same-origin' \
-H 'Accept-Language: en-GB,en-US;q=0.9,en;q=0.8' \
-H 'Accept-Encoding: gzip, deflate, br, zstd' \
-H 'Sec-Fetch-Mode: cors' \
-H 'Origin: https://scanner.darisini.com' \
-H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.5 Safari/605.1.15' \
-H 'Referer: https://scanner.darisini.com/v2/presence' \
-H 'Content-Length: 383' \
-H 'Sec-Fetch-Dest: empty' \
-H 'Cookie: __Secure-next-auth.session-token=eyJhbGciOiJkaXIiLCJlbmMiOiJBMjU2R0NNIn0..rHdx0zhM6BoPzvol.OlUR9i_qcFyrD4NHj3fxV6gxDmNOaSpYTAusiyZd1u5_0DXxybWa5qbk9px9JAp4tFJPWnFeICs3jw2-fUbU0K4f0t6Oq-7MO-f6HCR2vHCgjrsYaZABwnuO2zKtYAAKasMzyrGaupw0_HfldgQqlSyp5jAgn3qoP7ygAx-RtdnQjWh96A7QYAzxEEaMbftQfr0v1CQG380Zag0AWM6ODEsfYqYAoMSXJiPXnQxP-AxOFf_4PT-4NDmioB30Tvu3zyT9ZpZbcjnTgym2OGQ2uhjREFggVOdQ2NJB7xyVWTRTf7ZLjkoeDHZNwVwNVNOtz-IAZR_LECh_JinAwXe4Ihyf5wAZyQDdQmGWDIZ-AZXT_sHevwLtlqaUCgcLJxbM1dM2HA.Ork39Oq-EyrDo35kBt3pNg; __Secure-next-auth.callback-url=https%3A%2F%2Fscanner.darisini.com%2Fv2%2Flogin%3FscannerId%3Dcmt12rzyl013js601r4p5kwj5; __Host-next-auth.csrf-token=776b5941fc516b60be9bbc71d759b952e2bfabccf3b081009c2a2d540e5c1945%7Ca9044a6dd47018df46d745a93e6ce8b03b0e77ef6ef59c1078f88374f9b320b3; _ga_1V8MHJJ0V3=GS2.1.s1771292429$o44$g1$t1771292648$j42$l0$h0; ph_phc_rX96fU8eX4FOrxRG71X0wQ0qTeS3C3X1l7EHl7HRAty_posthog=%7B%22%24device_id%22%3A%22019bafe4-a007-7cc1-8ddb-50529ca56569%22%2C%22distinct_id%22%3A%22019bafe4-a007-7cc1-8ddb-50529ca56569%22%2C%22%24sesid%22%3A%5B1771292648218%2C%22019c6944-e327-791d-92db-385733279e88%22%2C1771292648218%5D%2C%22%24initial_person_info%22%3A%7B%22r%22%3A%22%24direct%22%2C%22u%22%3A%22https%3A%2F%2Fdarisini.com%2F%22%7D%7D; _ga=GA1.1.2004572984.1768182556' \
-H 'Priority: u=3, i' \
--data-raw '{"query":"mutation useEventScannerCreateEventAttendanceMutation(\n  $input: EventScannerCreateEventAttendanceInput!\n) {\n  eventScannerCreateEventAttendance(input: $input) {\n    id\n    decodedId\n  }\n}\n","variables":{"input":{"publicId":"MLV8K7QP","eventScannerId":"cmt12rzyl013js601r4p5kwj5","identityMatch":"Ticket matches identity","scannerUserFullName":"Rijal","notes":""}}}'

