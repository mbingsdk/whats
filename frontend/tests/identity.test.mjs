import {test} from "node:test";
import assert from "node:assert/strict";
import {request,IdentityError,allPages} from "../lib/identity-api.ts";

test("identity mutations obtain CSRF, retain same-origin credentials and send current revision",async(t)=>{
 const calls=[];
 t.mock.method(globalThis,"fetch",async(path,options)=>{
  calls.push({path,options});
  return Response.json(path.endsWith("/csrf")?{csrf_token:"synthetic-csrf"}:{ok:true});
 });
 await request("/api/v1/organizations/synthetic/members","POST",{name:"synthetic"},7);
 assert.equal(calls.length,2);
 assert.equal(calls[1].options.headers["X-CSRF-Token"],"synthetic-csrf");
 assert.equal(calls[1].options.headers["If-Match"],'"7"');
 assert.equal(calls[1].options.credentials,"same-origin");
 assert.equal(calls[1].options.cache,"no-store");
});
test("identity errors expose only a stable code and never server response details",async(t)=>{
 t.mock.method(globalThis,"fetch",async()=>Response.json({error:{code:"PERMISSION_DENIED",message:"private database detail"}},{status:403}));
 await assert.rejects(request("/api/v1/organization"),e=>e instanceof IdentityError&&e.code==="PERMISSION_DENIED"&&!e.message.includes("private"));
 await assert.rejects(request("https://outside.invalid/api/v1/"),/Invalid API path/);
});
test("all-page selection follows opaque cursors without losing rows",async(t)=>{
 const paths=[];
 t.mock.method(globalThis,"fetch",async path=>{paths.push(path);return Response.json(paths.length===1?{items:[{id:"one"}],next_cursor:"signed.cursor",has_more:true}:{items:[{id:"two"}],next_cursor:null,has_more:false});});
 assert.deepEqual(await allPages("/api/v1/security/sessions"),[{id:"one"},{id:"two"}]);
 assert.equal(paths[1],"/api/v1/security/sessions?cursor=signed.cursor");
});
