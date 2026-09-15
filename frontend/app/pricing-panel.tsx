"use client";
import Link from "next/link";
import {useCallback,useEffect,useState} from "react";
import {request} from "../lib/identity-api";
import {type Organization} from "./ui";
type Import={id:string;state:string;revision:number;source_url:string;source_sha256:string;import_hash:string;retrieved_at:string;next_review_at:string;validation_report:unknown;diff_report:unknown;reviewer_member_id:string|null};
type Policy={id:string;version:string;kind:string;source_url:string;source_sha256:string;review_evidence:string;effective_from:string;effective_to:string;next_review_at:string};
type Publication={id:string;snapshot_hash:string;published_at:string;next_review_at:string;supersedes_publication_id:string|null};
type Pricing={imports:Import[];policies:Policy[];publications:Publication[];settings:{billing_currency_state:string;billing_currency:string|null}[]};
type Budget={id:string;name:string;currency:string;hard_limit:string;exposure:string;starts_at:string;ends_at:string};
export default function PricingPanel({org}:{org:Organization}){
 const [correction,setCorrection]=useState("");
 const [data,setData]=useState<Pricing|null>(null),[budgets,setBudgets]=useState<Budget[]>([]),[error,setError]=useState(""),[busy,setBusy]=useState(false),[sending,setSending]=useState<{sending_enabled:boolean;policy_revision:number}|null>(null);
 const can=(permission:string)=>org.permissions.includes(permission);
 const reload=useCallback(async()=>{const [p,b]=await Promise.all([request<Pricing>("/api/v1/pricing"),request<Budget[]>("/api/v1/budgets")]);setData(p);setBudgets(b);},[]);
 useEffect(()=>{void Promise.resolve().then(reload).catch(e=>setError(e.message));if(org.permissions.includes("sending.manage"))void request<{settings:{sending_enabled:boolean;policy_revision:number}[]}>("/api/v1/inbox").then(v=>setSending(v.settings[0])).catch(e=>setError(e.message));},[reload,org.permissions]);
 const act=async(fn:()=>Promise<void>)=>{setBusy(true);setError("");try{await fn();await reload();}catch(e){setError(e instanceof Error?e.message:"Request failed.");}finally{setBusy(false);}};
 const action=(row:Import,verb:string,approve?:boolean)=>act(async()=>{await request("/api/v1/rate-cards/imports/"+row.id+"/"+verb,"POST",{revision:row.revision,...(approve===undefined?{}:{approve})});});
 return <section className="pricing-admin">
 <p className="notice"><strong>Paid-send authority is CLOSED.</strong> Company Meta billing currency: {data?.settings[0]?.billing_currency??"UNKNOWN"}. Rate imports do not establish the account&apos;s currency. Template and paid sends remain blocked.</p>
 <p>Registry changes require recent authentication and MFA. <Link href="/security">Review your security session</Link>.</p>
 {error&&<p className="error" role="alert">{error}</p>}
 {sending&&can("sending.manage")&&<section><h2>Sending switch</h2><p>{sending.sending_enabled?"Enabled; all other guard checks still apply.":"Disabled; all replies are blocked."}</p><button disabled={busy} onClick={()=>void act(async()=>{await request("/api/v1/inbox/settings","PATCH",{sending_enabled:!sending.sending_enabled,revision:sending.policy_revision});const result=await request<{settings:{sending_enabled:boolean;policy_revision:number}[]}>("/api/v1/inbox");setSending(result.settings[0]);})}>{sending.sending_enabled?"Disable sending":"Enable guarded sending"}</button></section>}
 <section><h2>Published Service policies</h2>{data?.policies.map(p=><article key={p.id}><h3>{p.version}</h3><p>{p.kind} · {new Date(p.effective_from).toLocaleString()} to {new Date(p.effective_to).toLocaleString()}</p><p>Review due: {new Date(p.next_review_at).toLocaleString()}</p><p>{p.review_evidence}</p><a href={p.source_url} target="_blank" rel="noreferrer">Official pricing source</a><details><summary>Source fingerprint</summary><code>{p.source_sha256}</code></details></article>)}</section>
 <section><h2>Rate Card Registry</h2><p>Import the reviewed July/October Indonesia artifacts in USD and IDR as independent coverage. Publishing an account rate selection requires verified billing currency and a separate reviewer.</p>
 {can("pricing.registry.import")&&<label>Correction rationale (required when replacing a publication)<input value={correction} onChange={e=>setCorrection(e.target.value)} maxLength={1000}/></label>}
 {can("pricing.registry.import")&&<button disabled={busy} onClick={()=>void act(async()=>{await request("/api/v1/rate-cards/imports","POST",{source:"GATE_C_20260914",correction_reason:correction});})}>Import reviewed Gate C artifacts</button>}
 {data?.imports.length===0&&<p>No rate imports.</p>}
 {data?.imports.map(row=><article key={row.id} className="rate-import"><h3>Import {row.id.slice(0,8)} · {row.state}</h3><p>Revision {row.revision} · Review due {new Date(row.next_review_at).toLocaleString()}</p><details><summary>Evidence, validation and diff</summary><a href={row.source_url} target="_blank" rel="noreferrer">Official pricing source</a><p>Bundle SHA-256: <code>{row.source_sha256}</code></p><p>Canonical import: <code>{row.import_hash}</code></p><pre>{JSON.stringify({validation:row.validation_report,diff:row.diff_report},null,2)}</pre></details><div className="actions">
 {can("pricing.registry.import")&&row.state==="DRAFT"&&<button disabled={busy} onClick={()=>void action(row,"validate")}>Validate</button>}
 {can("pricing.registry.import")&&row.state==="VALIDATED"&&<button disabled={busy} onClick={()=>void action(row,"diff")}>Compute diff</button>}
 {can("pricing.registry.import")&&row.state==="DIFFED"&&<button disabled={busy} onClick={()=>void action(row,"submit")}>Request review</button>}
 {can("pricing.registry.review")&&row.state==="IN_REVIEW"&&<><button disabled={busy} onClick={()=>void action(row,"review",true)}>Approve reviewed revision</button><button disabled={busy} onClick={()=>void action(row,"review",false)}>Request changes</button></>}
 {can("pricing.registry.import")&&row.state==="CHANGES_REQUIRED"&&<button disabled={busy} onClick={()=>void action(row,"revise")}>Start revised review</button>}
 {can("pricing.registry.publish")&&row.state==="APPROVED"&&<button disabled={busy} onClick={()=>void action(row,"publish")}>Publish immutable revision</button>}
 </div></article>)}</section>
 <section><h2>Publication history</h2>{!data?.publications.length&&<p>No numeric rate publication. Reviewed zero-cost Service policy is independent.</p>}{data?.publications.map(p=><p key={p.id}>{new Date(p.published_at).toLocaleString()} · <code>{p.snapshot_hash}</code>{p.supersedes_publication_id?" · Supersedes "+p.supersedes_publication_id:""}</p>)}</section>
 <section><h2>Budget exposure</h2><p>Zero-cost replies create no monetary reservation or ledger entry. Paid budgets require a verified currency; they do not grant permission to send paid messages.</p>
 {budgets.map(b=><p key={b.id}>{b.name} · {b.exposure} / {b.hard_limit} {b.currency} · {new Date(b.ends_at).toLocaleString()}</p>)}
 {can("budgets.manage")&&<form onSubmit={e=>{e.preventDefault();const f=new FormData(e.currentTarget);void act(async()=>{await request("/api/v1/budgets","POST",{Name:String(f.get("name")),Currency:String(f.get("currency")),Maximum:String(f.get("maximum")),Starts:new Date(String(f.get("starts"))).toISOString(),Ends:new Date(String(f.get("ends"))).toISOString()});});}}>
 <label>Budget name<input name="name" required maxLength={80}/></label><label>Verified currency<input name="currency" required pattern="[A-Z]{3}" maxLength={3} placeholder="Unknown until verified"/></label><label>Hard limit (decimal)<input name="maximum" required inputMode="decimal" pattern="[0-9]+(\.[0-9]{1,8})?" placeholder="0.00"/></label><label>Starts<input type="datetime-local" name="starts" required/></label><label>Ends<input type="datetime-local" name="ends" required/></label><button disabled={busy||data?.settings[0]?.billing_currency_state!=="VERIFIED"}>Create budget</button></form>}
 </section></section>;
}
