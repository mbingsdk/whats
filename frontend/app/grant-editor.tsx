"use client";
import {useState} from "react";
import {type Row,type Grant} from "./ui";
export default function GrantEditor({member,roles,teams,busy,save}:{member:Row;roles:Row[];teams:Row[];busy:boolean;save:(grants:Grant[])=>Promise<void>}){
 const [grants,setGrants]=useState<Grant[]>(member.grants??[]);
 function update(index:number,change:Partial<Grant>){setGrants(old=>old.map((g,i)=>i===index?{...g,...change}:g));}
 return <form onSubmit={e=>{e.preventDefault();void save(grants);}}>
 <p>Organization grants can be combined with team or self grants.</p>
 {grants.map((g,i)=><fieldset key={i}><legend>Grant {i+1}</legend>
 <label>Role<select required value={g.role_id} onChange={e=>update(i,{role_id:e.target.value})}><option value="">Choose a role</option>{roles.map(r=><option key={r.id} value={r.id}>{r.name}</option>)}</select></label>
 <label>Scope<select value={g.scope} onChange={e=>update(i,{scope:e.target.value as Grant["scope"],team_id:null})}><option value="ORG">Organization</option><option value="TEAM">Team</option><option value="SELF">Self</option></select></label>
 {g.scope==="TEAM"&&<label>Team<select required value={g.team_id??""} onChange={e=>update(i,{team_id:e.target.value})}><option value="">Choose team</option>{teams.filter(t=>!t.archived_at).map(t=><option key={t.id} value={t.id}>{t.name}</option>)}</select></label>}
 <button type="button" disabled={busy} onClick={()=>setGrants(old=>old.filter((_,n)=>n!==i))}>Remove grant</button>
 </fieldset>)}
 <button type="button" disabled={busy||grants.length>=20} onClick={()=>setGrants(old=>[...old,{role_id:"",scope:"ORG",team_id:null}])}>Add grant</button>
 <button disabled={busy} type="submit">Replace grants</button>
 </form>;
}
