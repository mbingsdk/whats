"use client";
import {useEffect,useState} from "react";
import {request,allPages} from "../lib/identity-api";
import {Field,Form,text,confirmAction,type User,type Row,type Act} from "./ui";
export default function Security({user,busy,act,refresh,version}:{user:User;busy:boolean;act:Act;refresh:()=>Promise<void>;version:number}){
 const [sessions,setSessions]=useState<Row[]>([]),[secret,setSecret]=useState(""),[codes,setCodes]=useState<string[]>([]),[error,setError]=useState("");
 useEffect(()=>{void allPages<Row>("/api/v1/security/sessions").then(setSessions).catch(e=>setError(e.message));},[version]);
 return <>{error&&<p role="alert">{error}</p>}
 <section><h2>Sensitive re-authentication</h2><p>Confirm your identity before sensitive changes. Confirmation lasts five minutes.</p>
 <Form label="Confirm identity" busy={busy} submit={d=>act(async()=>{await request("/api/v1/auth/reauthenticate","POST",{password:text(d,"password"),code:text(d,"code")});await refresh();},"Identity confirmed for five minutes.")}>
 <Field name="password" label="Current password" type="password" autoComplete="current-password"/>{user.mfa_enabled&&<Field name="code" label="Authenticator or recovery code" autoComplete="one-time-code"/>}</Form></section>
 <section><h2>Two-factor authentication</h2><p>Status: {user.mfa_enabled?"Enabled":"Not enrolled"}</p>
 {!user.mfa_enabled&&!secret&&<button disabled={busy} onClick={()=>void act(async()=>{const r=await request<{secret:string}>("/api/v1/auth/mfa/enrollment","POST",{});setSecret(r.secret);},"Add the secret to your authenticator, then enter its code.")}>Set up authenticator</button>}
 {secret&&<><label>Authenticator setup secret<input readOnly value={secret} aria-describedby="secret-help"/></label><p id="secret-help">Shown only during setup. Keep it private.</p>
 <Form label="Enable MFA" busy={busy} submit={d=>act(async()=>{const r=await request<{recovery_codes:string[]}>("/api/v1/auth/mfa/enrollment/confirm","POST",{code:text(d,"code")});setCodes(r.recovery_codes);setSecret("");await refresh();},"MFA enabled. Store your recovery codes safely.")}><Field name="code" label="Six-digit authenticator code" autoComplete="one-time-code"/></Form></>}
 {user.mfa_enabled&&<div className="actions"><button disabled={busy} onClick={()=>{if(confirmAction("Regenerate recovery codes","All previous recovery codes will stop working."))void act(async()=>{const r=await request<{result:{recovery_codes:string[]}}>("/api/v1/auth/mfa/recovery-codes/regenerate","POST",{});setCodes(r.result.recovery_codes);await refresh();});}}>Regenerate recovery codes</button>
 <button className="danger" disabled={busy} onClick={()=>{if(confirmAction("Disable MFA","Other sessions will become invalid. Your account will no longer require an authenticator at login."))void act(async()=>{await request("/api/v1/auth/mfa/disable","POST",{});setCodes([]);await refresh();});}}>Disable MFA</button></div>}
 {codes.length>0&&<div><h3>Recovery codes - shown once</h3><p>Store these privately. Each code works once.</p><textarea readOnly value={codes.join("\n")} rows={10} aria-label="Recovery codes"/><button onClick={()=>setCodes([])}>I have saved my codes</button></div>}
 </section>
 <section><h2>Active sessions</h2><button disabled={busy} onClick={()=>{if(confirmAction("Revoke other sessions","All other signed-in sessions for your account will be invalidated."))void act(async()=>{await request("/api/v1/security/sessions/revoke-others","POST",{});await refresh();});}}>Revoke all other sessions</button>
 <table><thead><tr><th>Session</th><th>Created</th><th>Expires</th><th>Action</th></tr></thead><tbody>{sessions.map(r=><tr key={r.id}><td>{r.current?"This session":r.id.slice(0,8)}</td><td>{r.created_at}</td><td>{r.expires_at}</td><td><button disabled={busy} onClick={()=>{if(confirmAction("Revoke session "+r.id.slice(0,8),"This session will be signed out."))void act(async()=>{await request("/api/v1/security/sessions/"+r.id+"/revoke","POST",{});await refresh();});}}>Revoke</button></td></tr>)}</tbody></table></section>
 </>;
}
