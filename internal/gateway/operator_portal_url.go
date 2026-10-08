package gateway

import (
 "net/url"
 "os"
 "strings"
)

func operatorApprovalURL(requestID any) string {
 id, ok := requestID.(string)
 if !ok || !strings.HasPrefix(id,"apr_") || len(id) > 104 { return "" }
 for _, c := range id { if !((c>='a'&&c<='z')||(c>='A'&&c<='Z')||(c>='0'&&c<='9')||c=='_'||c=='-') { return "" } }
 root := os.Getenv("VPS_AGENT_OPERATOR_PORTAL_URL")
 u,err:=url.Parse(root)
 if err!=nil || u.Scheme!="https" || u.Host=="" || u.User!=nil || u.RawQuery!="" || u.Fragment!="" {return ""}
 q:=u.Query();q.Set("request",id);u.RawQuery=q.Encode()
 return u.String()
}
