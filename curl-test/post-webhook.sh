#!/bin/bash

#MEMBER1=99691212
#MEMBER2=99691212
MEMBER1=99999999
MEMBER2=88888888
SUBMISSIONID=$(date +%s%3N)
SESSION_DATE1=2026-02-01
SESSION_DATE2=2026-02-02

curl -X POST https://rvpzpqjytyw6pilrg4glui2ko40qudxj.lambda-url.eu-west-3.on.aws/ -d "--------------------------CjIexjxdCcNhzczUt4wo08
Content-Disposition: form-data; name=\"action\"


--------------------------CjIexjxdCcNhzczUt4wo08
Content-Disposition: form-data; name=\"webhookURL\"

https://rvpzpqjytyw6pilrg4glui2ko40qudxj.lambda-url.eu-west-3.on.aws/
--------------------------CjIexjxdCcNhzczUt4wo08
Content-Disposition: form-data; name=\"username\"

bathridingclub
--------------------------CjIexjxdCcNhzczUt4wo08
Content-Disposition: form-data; name=\"formID\"

252725624662359
--------------------------CjIexjxdCcNhzczUt4wo08
Content-Disposition: form-data; name=\"type\"

WEB
--------------------------CjIexjxdCcNhzczUt4wo08
Content-Disposition: form-data; name=\"customParams\"


--------------------------CjIexjxdCcNhzczUt4wo08
Content-Disposition: form-data; name=\"product\"


--------------------------CjIexjxdCcNhzczUt4wo08
Content-Disposition: form-data; name=\"formTitle\"

Training
--------------------------CjIexjxdCcNhzczUt4wo08
Content-Disposition: form-data; name=\"customTitle\"


--------------------------CjIexjxdCcNhzczUt4wo08
Content-Disposition: form-data; name=\"submissionID\"

${SUBMISSIONID}
--------------------------CjIexjxdCcNhzczUt4wo08
Content-Disposition: form-data; name=\"event\"


--------------------------CjIexjxdCcNhzczUt4wo08
Content-Disposition: form-data; name=\"documentID\"


--------------------------CjIexjxdCcNhzczUt4wo08
Content-Disposition: form-data; name=\"teamID\"


--------------------------CjIexjxdCcNhzczUt4wo08
Content-Disposition: form-data; name=\"subject\"


--------------------------CjIexjxdCcNhzczUt4wo08
Content-Disposition: form-data; name=\"isSilent\"


--------------------------CjIexjxdCcNhzczUt4wo08
Content-Disposition: form-data; name=\"customBody\"


--------------------------CjIexjxdCcNhzczUt4wo08
Content-Disposition: form-data; name=\"rawRequest\"

{\"slug\":\"submit\/252725624662359\",\"jsExecutionTracker\":\"build-date-1767092416391=>init-started:1767093629905=>validator-called:1767093629928=>validator-mounted-false:1767093629928=>init-complete:1767093629930=>interval-complete:1767093650929=>onsubmit-fired:1767093681178=>observerSubmitHandler_received-submit-event:1767093681179=>submit-validation-passed:1767093681193=>observerSubmitHandler_validation-passed-submitting-form:1767093681204\",\"submitSource\":\"form\",\"submitDate\":\"1767093681204\",\"buildDate\":\"1767092416391\",\"uploadServerUrl\":\"https:\/\/upload.jotform.com\/upload\",\"eventObserver\":\"1\",\"q15_brcMembership15\":\"${MEMBER1}\",\"q28_typeA28\":[\"Current Club Membership\"],\"q18_horseName18\":\"test1\",\"q5_selectSession\":{\"implementation\":\"new\",\"date\":\"${SESSION_DATE1} 18:00\",\"duration\":\"60\",\"timezone\":\"Europe\/London (GMT+01:00)\"},\"q34_selectedVenue\":\"West Wilts\",\"q31_amount\":\"21\",\"q58_totalAmount\":\"37\",\"q53_paymentRef\":\"VSHE\",\"q12_typeA\":\"VSHE\",\"q54_wwecnonmem\":\"26\",\"q55_wwecmem\":\"21\",\"q56_widnonmem\":\"20\",\"q57_widmem\":\"16\",\"q48_brcMembership15-2\":\"${MEMBER2}\",\"q49_typeA28-2\":[\"Current Club Membership\"],\"q50_horseName18-2\":\"test2\",\"q51_selectSession-2\":{\"implementation\":\"new\",\"date\":\"${SESSION_DATE2} 18:00\",\"duration\":\"60\",\"timezone\":\"Europe\/London (GMT+00:00)\"},\"q60_selectedVenue-2\":\"Widbrook\",\"q59_amount-2\":\"16\",\"timeToSubmit\":\"20\",\"preview\":\"true\",\"validatedNewRequiredFieldIDs\":\"{}\",\"visitedPages\":\"{}\",\"path\":\"\/submit\/252725624662359\"}
--------------------------CjIexjxdCcNhzczUt4wo08
Content-Disposition: form-data; name=\"fromTable\"


--------------------------CjIexjxdCcNhzczUt4wo08
Content-Disposition: form-data; name=\"appID\"


--------------------------CjIexjxdCcNhzczUt4wo08
Content-Disposition: form-data; name=\"pretty\"

British Riding Clubs Membership Number:${MEMBER1}, :Current Club Membership, Horse Name:test1, Select Session Date and Preferred Time:Thursday, Jan 01, 2026 06:00 PM-07:00 PM Europe/London (GMT+01:00), Selected Venue::West Wilts, Amount (£)::21, Total Amount (£)::37, Payment Ref::VSHE, :VSHE, WWEC-NON-MEM:26, WWEC-MEM:21, WID-NON-MEM:20, WID-MEM:16, British Riding Clubs Membership Number:${MEMBER2}, :Current Club Membership, Horse Name:test2, Select Session Date and Preferred Time:Friday, Jan 02, 2026 06:00 PM-07:00 PM Europe/London (GMT+00:00), Selected Venue::Widbrook, Amount (£)::16
--------------------------CjIexjxdCcNhzczUt4wo08
Content-Disposition: form-data; name=\"unread\"


--------------------------CjIexjxdCcNhzczUt4wo08
Content-Disposition: form-data; name=\"parent\"


--------------------------CjIexjxdCcNhzczUt4wo08
Content-Disposition: form-data; name=\"ip\"

109.152.150.51
--------------------------CjIexjxdCcNhzczUt4wo08--
"

