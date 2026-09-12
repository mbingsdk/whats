"use client";
import Link from "next/link";
import {useCallback,useEffect,useState} from "react";
import {request,type Page} from "../lib/identity-api";
import {confirmAction,type Act,type Organization} from "./ui";
export const metaScreens=["meta-connection","wabas","phone-numbers","business-profiles","webhook-events","meta-health"] as const;
type Row=Record<string,unknown>&{id?:string;state?:string;callback_key?:string};
type Health={connections:Row[];processing:Row[];runtime_configured:boolean};
const paths:Record<string,string>={"meta-connection":"connection",wabas:"wabas","phone-numbers":"phone-numbers","business-profiles":"business-profiles","webhook-events":"webhook-events","meta-health":"health"};
const labels:Record<string,string>={"meta-connection":"Meta Connection",wabas:"WABA Accounts","phone-numbers":"Phone Numbers","business-profiles":"Business Profile","webhook-events":"Webhook Events","meta-health":"Meta Health"};
const columns:Record<string,string[]>={
"meta-connection":["external_id","configured_waba_id","graph_version","credential_state","last_sync_at","last_error"],
wabas:["external_id","name","timezone_id","subscribed","lifecycle","synced_at"],
"phone-numbers":["external_id","name","display_number","quality","platform","code_verification_status","lifecycle","synced_at"],
"business-profiles":["phone_id","fields","graph_version","synced_at"],
"webhook-events":["received_at","event_class","state","attempt_count","last_error","request_id"],
};
const title=(key:string)=>key.replaceAll("_"," ");
function value(v:unknown){if(v===null||v===undefined||v==="")return "—";if(typeof v==="boolean")return v?"Yes":"No";if(typeof v==="object")return JSON.stringify(v);return String(v);}
function Table({rows,columns,action}:{rows:Row[];columns:string[];action?:(row:Row)=>React.ReactNode}){
return <div className="meta-table"><table><thead><tr>{columns.map(c=><th key={c}>{title(c)}</th>)}{action&&<th>Actions</th>}</tr></thead><tbody>{rows.map((r,i)=><tr key={r.id??i}>{columns.map(c=><td key={c} data-field={c}>{value(r[c])}</td>)}{action&&<td>{action(r)}</td>}</tr>)}</tbody></table>{rows.length===0&&<p>No records yet.</p>}</div>;
}
export default function MetaPanel({screen,org,busy,act}:{screen:string;org:Organization;busy:boolean;act:Act}){
const [items,setItems]=useState<Row[]>([]),[cursor,setCursor]=useState<string|null>(null),[health,setHealth]=useState<Health|null>(null),[runs,setRuns]=useState<Row[]>([]),[error,setError]=useState(""),[loading,setLoading]=useState(true),[detail,setDetail]=useState<Row|null>(null),[payload,setPayload]=useState<unknown>(null);
const allowed=(p:string)=>org.permissions.includes(p);
const canRead=org.permissions.includes(screen==="webhook-events"?"webhooks.view":"meta.view");
const endpoint="/api/v1/meta/"+paths[screen];
const load=useCallback(async()=>{
if(!canRead)return;
if(screen==="meta-health")setHealth(await request<Health>(endpoint));
else{const page=await request<Page<Row>>(endpoint);setItems(page.items);setCursor(page.next_cursor);}
if(screen==="meta-connection")setRuns((await request<Page<Row>>("/api/v1/meta/sync-runs")).items);
},[canRead,endpoint,screen]);
useEffect(()=>{let active=true;void Promise.resolve().then(load).catch(e=>{if(active)setError(e.message);}).finally(()=>{if(active)setLoading(false);});return()=>{active=false;};},[load,org.id]);
async function sync(){await request("/api/v1/meta/sync","POST",{});await load();}
return <>
<nav className="meta-nav" aria-label="Meta operations">{metaScreens.filter(s=>org.permissions.includes(s==="webhook-events"?"webhooks.view":"meta.view")).map(s=><Link key={s} href={"/"+s} aria-current={s===screen?"page":undefined}>{labels[s]}</Link>)}</nav>
{!canRead?<p>You do not have permission to view this operation.</p>:<>
<div className="actions"><button disabled={busy} onClick={()=>void act(load,"Refreshed.")}>Refresh</button>{allowed("meta.manage")&&screen!=="webhook-events"&&<button disabled={busy} onClick={()=>void act(sync,"Sync queued. Refresh to see the worker result.")}>Synchronize existing assets</button>}</div>
{loading&&<p role="status">Loading Meta records...</p>}{error&&<p className="error" role="alert">{error}</p>}
{screen==="meta-connection"&&<><p>Connect the App and WABA configured by your operator. Credentials stay on the server. This action binds existing assets to this organization.</p>{items.length===0&&allowed("meta.manage")&&<button disabled={busy} onClick={()=>{if(confirmAction("Connect configured Meta App","Bind the operator-configured App to "+org.name+"?"))void act(async()=>{await request("/api/v1/meta/connection","POST",{});await load();},"Connection bound.");}}>Connect configured App</button>}
{items.map(row=><p key={row.id}>Callback path: <code>/api/v1/meta/webhooks/{row.callback_key}</code></p>)}
<p>Changing the callback in Meta requires operator approval. An existing WABA subscription is inspected during sync.</p></>}
{screen==="phone-numbers"&&<p>Quality and code verification are reported by Meta. NOT_VERIFIED alone does not establish registration or connectivity status.</p>}
{screen==="business-profiles"&&<p>Only fields returned by Meta are shown. A missing field has not been inferred.</p>}
{screen==="meta-health"&&health?<><section><h2>Meta connectivity</h2><p>Runtime configuration: {health.runtime_configured?"configured":"not configured"}. Read access is verified only at the last successful sync.</p><Table rows={health.connections} columns={["graph_version","credential_state","last_sync_at","last_error"]}/></section><section><h2>Webhook reception and local processing</h2><p>Challenge time records a valid verification request; live Meta provenance is recorded separately in acceptance evidence.</p><Table rows={health.connections} columns={["challenge_verified_at","last_webhook_at"]}/><Table rows={health.processing} columns={["state","count"]}/></section></>:screen!=="meta-health"&&<Table rows={items} columns={columns[screen]} action={screen==="webhook-events"?(row)=><button disabled={busy} onClick={()=>void act(async()=>{setPayload(null);setDetail(await request<Row>(endpoint+"/"+row.id));},"Event loaded.")}>Inspect</button>:undefined}/>}
{cursor&&<button disabled={busy} onClick={()=>void act(async()=>{const page=await request<Page<Row>>(endpoint+"?cursor="+encodeURIComponent(cursor));setItems(v=>[...v,...page.items]);setCursor(page.next_cursor);},"More records loaded.")}>Load more</button>}
{screen==="meta-connection"&&<section><h2>Synchronization history</h2><Table rows={runs} columns={["created_at","state","attempt_count","error_code","finished_at"]}/></section>}
{detail&&screen==="webhook-events"&&<section><h2>Event {detail.id}</h2><p>State: {detail.state}. Replay generation: {value(detail.generation)}. Payload retained: {value(detail.payload_retained)}.</p>
<div className="actions">{allowed("webhooks.payload.view")&&detail.state!=="QUARANTINED"&&Boolean(detail.payload_retained)&&<button disabled={busy} onClick={()=>void act(async()=>setPayload(await request(endpoint+"/"+detail.id+"/payload")),"Redacted structure loaded.")}>Inspect redacted payload</button>}
{allowed("webhooks.replay")&&Boolean(detail.payload_retained)&&["PROCESSED","UNKNOWN","INVALID","DEAD_LETTER"].includes(detail.state??"")&&<button disabled={busy} onClick={()=>{if(confirmAction("Replay event "+detail.id,"Queue the original stored event for classification again? Existing facts remain deduplicated. Recent authentication is required."))void act(async()=>{await request(endpoint+"/"+detail.id+"/replay","POST",{});setDetail(null);await load();},"Replay queued.");}}>Replay stored event</button>}</div>
<h3>Processing attempts</h3><Table rows={(detail.attempts??[]) as Row[]} columns={["generation","attempt_no","result","error_code","parser_version","created_at"]}/>
<h3>Classified facts</h3><Table rows={(detail.facts??[]) as Row[]} columns={["event_class","waba_id","phone_id","uncertain_identity"]}/>
{payload!==null&&<pre className="meta-payload">{JSON.stringify(payload,null,2)}</pre>}</section>}
</>}
</>;
}
