"use client";
import Link from "next/link";
import {useRouter} from "next/navigation";
import {useCallback,useEffect,useState} from "react";
import {IdentityError,request} from "../lib/identity-api";
import {type User,type Organization,type Act} from "./ui";
import Inbox from "./inbox-panel";
import Pricing from "./pricing-panel";
import {invalidateDrafts} from "../lib/inbox-drafts";
import AuthForms from "./auth-forms";
import Security from "./security-panel";
import Management from "./management";
import MetaPanel,{metaScreens} from "./meta-panel";
export type Screen="inbox"|"pricing"|"login"|"forgot-password"|"reset-password"|"verify-email"|"accept-invitation"|"security"|"members"|"teams"|"roles"|"audit"|"meta-connection"|"wabas"|"phone-numbers"|"business-profiles"|"webhook-events"|"meta-health";
const titles:Record<Screen,string>={inbox:"Inbox",pricing:"Pricing & send policy",login:"Sign in","forgot-password":"Request password reset","reset-password":"Set a new password","verify-email":"Verify your email","accept-invitation":"Accept your invitation",security:"Account & security",members:"Members & invitations",teams:"Teams",roles:"Roles & permissions",audit:"Audit history","meta-connection":"Meta Connection",wabas:"WABA Accounts","phone-numbers":"Phone Numbers","business-profiles":"Business Profile","webhook-events":"Webhook Events","meta-health":"Meta Health"};
export default function Identity({screen}:{screen:Screen}){
 const router=useRouter();
 const [user,setUser]=useState<User|null>(null),[org,setOrg]=useState<Organization|null>(null),[organizations,setOrganizations]=useState<{id:string;name:string}[]>([]);
 const [loading,setLoading]=useState(true),[busy,setBusy]=useState(false),[message,setMessage]=useState(""),[error,setError]=useState(""),[version,setVersion]=useState(0);
 const privateScreen=["inbox","pricing","security","members","teams","roles","audit",...metaScreens].includes(screen);
 const refresh=useCallback(async()=>{
  try{
   const session=await request<{user:User;organizations:{id:string;name:string}[]}>("/api/v1/auth/session");
   setUser(session.user);setOrganizations(session.organizations);
   if(!session.user.organization_id&&session.organizations.length===1)await request("/api/v1/auth/organization-session","POST",{organization_id:session.organizations[0].id});
   if(session.user.organization_id||session.organizations.length===1)setOrg(await request<Organization>("/api/v1/organization"));
   setVersion(v=>v+1);
  }catch(e){if(e instanceof IdentityError&&e.status===401){setUser(null);if(privateScreen)router.push("/login");}else throw e;}
 },[privateScreen,router]);
 useEffect(()=>{let active=true;void Promise.resolve().then(refresh).catch(e=>{if(active)setError(e.message);}).finally(()=>{if(active)setLoading(false);});return ()=>{active=false;};},[refresh]);
 const act:Act=async(fn,success="Saved.")=>{
  setBusy(true);setError("");setMessage("");
  try{await fn();setMessage(success);}catch(e){setError(e instanceof Error?e.message:"Request failed.");}finally{setBusy(false);}
 };
 if(loading)return <main><p role="status">Loading account...</p></main>;
 const allowed=(p:string)=>org?.permissions.includes(p)??false;
 return <div className="app"><header><Link href="/security" className="brand">WABA Control</Link>{user&&<><span>{user.name} · {user.email}</span><button disabled={busy} onClick={()=>void act(async()=>{await request("/api/v1/auth/logout","POST",{});invalidateDrafts();router.push("/login");})}>Sign out</button></>}</header>
 {user&&<nav aria-label="Application">{(allowed("inbox.view")||org?.self_permissions?.includes("inbox.view")||org?.scoped_teams.some(t=>t.permissions.includes("inbox.view")))&&<Link href="/inbox">Inbox</Link>}{allowed("pricing.view")&&<Link href="/pricing">Pricing</Link>}<Link href="/security">Account & security</Link>{allowed("members.view")&&<Link href="/members">Members</Link>}{(allowed("teams.view")||(org?.scoped_teams.length??0)>0)&&<Link href="/teams">Teams</Link>}{allowed("roles.view")&&<Link href="/roles">Roles</Link>}{allowed("audit.view")&&<Link href="/audit">Audit</Link>}{allowed("meta.view")&&<Link href="/meta-connection">Meta</Link>}{allowed("webhooks.view")&&!allowed("meta.view")&&<Link href="/webhook-events">Webhook Events</Link>}</nav>}
 <main className={privateScreen?"workspace":"identity"}><h1>{titles[screen]}</h1>
 {error&&<p className="error" role="alert">{error}. {privateScreen&&<Link href="/security">Review authentication and permissions.</Link>}</p>}
 {message&&<p className="notice" role="status">{message}</p>}
 {user&&!user.verified&&<p className="notice">Verify your email before managing organization access. <Link href="/verify-email">Email verification</Link></p>}
 {organizations.length>1&&<label>Organization<select value={org?.id??""} onChange={e=>void act(async()=>{await request("/api/v1/auth/organization-session","POST",{organization_id:e.target.value});invalidateDrafts();await refresh();})}><option value="">Select organization</option>{organizations.map(o=><option key={o.id} value={o.id}>{o.name}</option>)}</select></label>}
 {!privateScreen&&<AuthForms screen={screen} user={user} busy={busy} act={act} refresh={refresh}/>}
 {screen==="security"&&user&&<Security user={user} busy={busy} act={act} refresh={refresh} version={version}/>}
 {["members","teams","roles","audit"].includes(screen)&&org&&<Management screen={screen} org={org} busy={busy} act={act} refresh={refresh} version={version}/>}
 {metaScreens.some(s=>s===screen)&&org&&<MetaPanel key={org.id+screen} screen={screen} org={org} busy={busy} act={act}/>}
 {screen==="inbox"&&org&&user&&<Inbox key={org.id+user.id} org={org} user={user}/>}
 {screen==="pricing"&&org&&<Pricing org={org}/>}
 </main></div>;
}
