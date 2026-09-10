"use client";
import {useEffect,useState} from "react";
import GrantEditor from "./grant-editor";
import {request,allPages,type Page} from "../lib/identity-api";
import {Field,Form,text,confirmAction,type Organization,type Row,type Act} from "./ui";
const keys=["organization.view","members.view","members.invite","members.manage","teams.view","teams.manage","roles.view","roles.manage","roles.assign","owners.manage","audit.view"];
export default function Management({screen,org,busy,act,refresh,version}:{screen:string;org:Organization;busy:boolean;act:Act;refresh:()=>Promise<void>;version:number}){
 const [rows,setRows]=useState<Row[]>([]),[roles,setRoles]=useState<Row[]>([]),[members,setMembers]=useState<Row[]>([]),[invites,setInvites]=useState<Row[]>([]),[error,setError]=useState("");
 const [cursor,setCursor]=useState<string|null>(null);
 const [grantTeams,setGrantTeams]=useState<Row[]>([]);
 const [team,setTeam]=useState<Row|null>(null),[teamMembers,setTeamMembers]=useState<Row[]>([]);
 const base="/api/v1/organizations/"+org.id+"/";
 const allowed=(p:string)=>org.permissions.includes(p);
 const inTeam=(p:string,id:string)=>allowed(p)||org.scoped_teams.some(t=>t.id===id&&t.permissions.includes(p));
 useEffect(()=>{
  let current=true;
  async function load(){
   const permission=screen==="members"?"members.view":screen==="teams"?"teams.view":screen==="roles"?"roles.view":"audit.view";
   if(!org.permissions.includes(permission)&&!(screen==="teams"&&org.scoped_teams.length>0))return;
   const result=await request<Page<Row>>(base+screen);if(!current)return;setRows(result.items);setCursor(result.next_cursor);
   if(["members","teams"].includes(screen)&&org.permissions.includes("roles.view"))setRoles(await allPages<Row>(base+"roles"));
   if(screen==="teams"&&org.permissions.includes("members.view"))setMembers(await allPages<Row>(base+"members"));
   if(screen==="members"&&org.permissions.includes("teams.view"))setGrantTeams(await allPages<Row>(base+"teams"));
   if(screen==="members"&&org.permissions.includes("members.invite"))setInvites(await allPages<Row>(base+"invitations"));
  }
  void load().catch(e=>{if(current)setError(e.message);});
  return ()=>{current=false;};
 },[base,org.permissions,org.scoped_teams,screen,version]);
 async function mutate(path:string,method:string,body:unknown,revision?:number){await request(base+path,method,body,revision);await refresh();}
 function roleSelect(){return <label>Role<select name="role_id" required><option value="">Choose a role</option>{roles.map(r=><option key={r.id} value={r.id}>{r.name}</option>)}</select></label>;}
 const access=screen==="members"?"members.view":screen==="teams"?"teams.view":screen==="roles"?"roles.view":"audit.view";
 if(!allowed(access)&&!(screen==="teams"&&org.scoped_teams.length>0))return <p role="status">You do not have permission to view this page.</p>;
 return <>{error&&<p role="alert">{error}</p>}
 {screen==="members"&&<>
 <section><h2>Organization members</h2><table><thead><tr><th>Name / email</th><th>Status</th><th>Access</th></tr></thead><tbody>{rows.map(r=><tr key={r.id}><td>{r.display_name}<small>{r.canonical_email}</small></td><td>{r.status}</td><td>
 {allowed("members.manage")&&<button disabled={busy} className={r.status==="ACTIVE"?"danger":""} onClick={()=>{const action=r.status==="ACTIVE"?"deactivate":"reactivate";if(confirmAction(action+" "+r.display_name,action==="deactivate"?"Organization access stops immediately and team memberships are removed.":"Organization access is restored with existing role grants."))void act(()=>mutate("members/"+r.id+"/"+action,"POST",{},r.revision));}}>{r.status==="ACTIVE"?"Deactivate":"Reactivate"}</button>}
 {allowed("roles.assign")&&<details><summary>Change role</summary><GrantEditor key={r.id+":"+r.revision} member={r} roles={roles} teams={grantTeams} busy={busy} save={grants=>act(async()=>{if(confirmAction("Replace grants for "+r.display_name,"Access changes invalidate affected organization sessions."))await mutate("members/"+r.id+"/role-grants","PUT",{grants},r.revision);})}/></details>}</td></tr>)}</tbody></table></section>
 {allowed("members.invite")&&<section><h2>Invite an employee</h2><Form label="Send invitation" busy={busy} submit={d=>act(()=>mutate("invitations","POST",{email:text(d,"email"),role_ids:[text(d,"role_id")]}),"Invitation queued.")}><Field name="email" label="Employee email" type="email" autoComplete="email"/>{roleSelect()}</Form>
 <h3>Invitations</h3><table><thead><tr><th>Email</th><th>Status</th><th>Actions</th></tr></thead><tbody>{invites.map(r=><tr key={r.id}><td>{r.canonical_email}</td><td>{r.state}<small>Mail: {r.delivery_state??"Pending"}</small></td><td>{r.state==="PENDING"&&<><button disabled={busy} onClick={()=>void act(()=>mutate("invitations/"+r.id+"/resend","POST",{},r.revision),"New link queued; previous link invalidated.")}>Resend</button><button disabled={busy} className="danger" onClick={()=>{if(confirmAction("Revoke invitation for "+r.canonical_email,"The invitation link stops working."))void act(()=>mutate("invitations/"+r.id+"/revoke","POST",{},r.revision));}}>Revoke</button></>}</td></tr>)}</tbody></table></section>}
 </>}
 {screen==="teams"&&<>
 {allowed("teams.manage")&&<Form label="Create team" busy={busy} submit={d=>act(()=>mutate("teams","POST",{name:text(d,"name")}))}><Field name="name" label="Team name"/></Form>}
 <table><thead><tr><th>Team</th><th>Status</th><th>Actions</th></tr></thead><tbody>{rows.map(r=><tr key={r.id}><td>{r.name}</td><td>{r.archived_at?"Archived":"Active"}</td><td><button disabled={busy} onClick={()=>void act(async()=>{setTeam(r);setTeamMembers(await allPages<Row>(base+"teams/"+r.id+"/members"));},"")}>Members</button>
 {inTeam("teams.manage",r.id)&&!r.archived_at&&<><details><summary>Rename</summary><Form label="Rename team" busy={busy} submit={d=>act(()=>mutate("teams/"+r.id,"PATCH",{name:text(d,"name")},r.revision))}><Field name="name" label="New team name"/></Form></details><button className="danger" disabled={busy} onClick={()=>{if(confirmAction("Archive "+r.name,"Inherited team permissions no longer apply."))void act(()=>mutate("teams/"+r.id+"/archive","POST",{},r.revision));}}>Archive</button></>}</td></tr>)}</tbody></table>
 {team&&<section><h2>{team.name} members</h2><ul>{teamMembers.map(m=><li key={m.member_id}>{m.display_name} {inTeam("teams.manage",team.id)&&<button disabled={busy} onClick={()=>void act(async()=>{const current=rows.find(r=>r.id===team.id)??team;await mutate("teams/"+team.id+"/members/"+m.member_id,"DELETE",{},current.revision);setTeam(null);})}>Remove</button>}</li>)}</ul>
 {inTeam("teams.manage",team.id)&&!team.archived_at&&<Form label="Add member" busy={busy} submit={d=>act(async()=>{const current=rows.find(r=>r.id===team.id)??team;await mutate("teams/"+team.id+"/members","POST",{member_id:text(d,"member_id")},current.revision);setTeam(null);})}><label>Member<select name="member_id" required><option value="">Choose member</option>{members.filter(m=>m.status==="ACTIVE").map(m=><option key={m.id} value={m.id}>{m.display_name}</option>)}</select></label></Form>}
 {inTeam("roles.assign",team.id)&&<Form label="Replace team role" busy={busy} submit={d=>act(async()=>{if(confirmAction("Change role for "+team.name,"Team members inherit this role within this team.")){const current=rows.find(r=>r.id===team.id)??team;await mutate("teams/"+team.id+"/role-grants","PUT",{role_ids:[text(d,"role_id")]},current.revision);setTeam(null);}})}>{roleSelect()}</Form>}
 </section>}
 </>}
 {screen==="roles"&&<><p>Permissions are explicit and constrained by the administrator&apos;s delegation authority.</p>
 {rows.map(r=><section key={r.id}><h2>{r.name}</h2><p>{r.permissions?.join(", ")||"No permissions"}</p>{allowed("roles.manage")&&<details><summary>Edit role</summary><Form label="Save role" busy={busy} submit={d=>act(async()=>{if(confirmAction("Change "+r.name,"Changes affect every member and team assigned this role."))await mutate("roles/"+r.id,"PATCH",{name:text(d,"name"),permissions:d.getAll("permission")},r.revision);})}><label>Role name<input name="name" required defaultValue={r.name}/></label><fieldset><legend>Permissions</legend>{keys.map(k=><label className="check" key={k}><input type="checkbox" name="permission" value={k} defaultChecked={r.permissions?.includes(k)}/>{k}</label>)}</fieldset></Form></details>}</section>)}
 {allowed("roles.manage")&&<section><h2>Create custom role</h2><Form label="Create role" busy={busy} submit={d=>act(()=>mutate("roles","POST",{name:text(d,"name"),permissions:d.getAll("permission")}))}><Field name="name" label="Role name"/><fieldset><legend>Permissions</legend>{keys.map(k=><label className="check" key={k}><input type="checkbox" name="permission" value={k}/>{k}</label>)}</fieldset></Form></section>}</>}
 {screen==="audit"&&<table><thead><tr><th>Time</th><th>Action</th><th>Resource</th></tr></thead><tbody>{rows.map(r=><tr key={r.id}><td>{r.created_at}</td><td>{r.action}</td><td>{r.resource_id}</td></tr>)}</tbody></table>}
 {cursor&&<button disabled={busy} onClick={()=>void act(async()=>{const result=await request<Page<Row>>(base+screen+"?cursor="+encodeURIComponent(cursor));setRows(old=>[...old,...result.items]);setCursor(result.next_cursor);},"")}>Load more</button>}
 </>;
}
