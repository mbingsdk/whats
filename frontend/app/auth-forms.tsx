"use client";
import Link from "next/link";
import {useRouter} from "next/navigation";
import {useEffect,useState} from "react";
import {request} from "../lib/identity-api";
import {Field,Form,text,type Act,type User} from "./ui";
export default function AuthForms({screen,user,busy,act,refresh}:{screen:string;user:User|null;busy:boolean;act:Act;refresh:()=>Promise<void>}){
 const router=useRouter();
 const [proof,setProof]=useState(()=>typeof window==="undefined"?"":new URLSearchParams(window.location.hash.slice(1)).get("token")??""),[challenge,setChallenge]=useState("");
 useEffect(()=>{if(window.location.hash)window.history.replaceState(null,"",window.location.pathname);},[]);
 return <>
 {screen==="login"&&<><p>Use your company invitation or existing account.</p><Form label={challenge?"Verify and sign in":"Sign in"} busy={busy} submit={d=>act(async()=>{
  const r=await request<{mfa_required?:boolean;challenge?:string}>("/api/v1/auth/"+(challenge?"mfa/verify":"login"),"POST",challenge?{token:challenge,code:text(d,"code")}:{email:text(d,"email"),password:text(d,"password")});
  if(r.mfa_required){setChallenge(r.challenge??"");return;}router.push("/security");
 },challenge?"Signed in.":"Continue with your authenticator if requested.")}>
 {!challenge?<><Field name="email" label="Email" type="email" autoComplete="username"/><Field name="password" label="Password" type="password" autoComplete="current-password"/></>:<Field name="code" label="Authenticator or recovery code" autoComplete="one-time-code"/>}
 </Form><Link href="/forgot-password">Forgot password?</Link></>}
 {(screen==="forgot-password"||(screen==="verify-email"&&!proof))&&<Form label="Send email" busy={busy} submit={d=>act(async()=>{await request("/api/v1/auth/"+(screen==="forgot-password"?"password-reset":"email-verification")+"/request","POST",{email:text(d,"email")});},"If the address is eligible, an email will be sent. Check your inbox.")}><Field name="email" label="Email" type="email" autoComplete="email"/></Form>}
 {screen==="verify-email"&&proof&&<Form label="Verify email" busy={busy} submit={()=>act(async()=>{await request("/api/v1/auth/email-verification/complete","POST",{token:proof});setProof("");await refresh();},"Email verified. You can sign in.")}><p>Confirm verification of the address linked to this email.</p></Form>}
 {(screen==="reset-password"||screen==="accept-invitation")&&<Form label={screen==="reset-password"?"Reset password":"Accept invitation"} busy={busy} submit={d=>act(async()=>{
  if(!proof)throw new Error("Open the action link from your email");
  await request(screen==="reset-password"?"/api/v1/auth/password-reset/complete":"/api/v1/invitations/accept","POST",{token:proof,...(screen==="reset-password"||!user?{password:text(d,"password")}:{}),...(screen==="accept-invitation"?{name:text(d,"name")}:{})});
  setProof("");router.push("/login");
 })}>
 {!proof&&<p role="alert">Open the link from your email. Links expire and can be used once.</p>}
 {screen==="accept-invitation"&&!user&&<Field name="name" label="Your name" autoComplete="name"/>}
 {(screen==="reset-password"||!user)&&<Field name="password" label="New password (12–256 characters)" type="password" autoComplete="new-password" minLength={12}/>}
 {screen==="accept-invitation"&&<p>Already have an account? <Link href="/login" target="_blank">Sign in in another tab</Link>, then reopen the original invitation link. An invitation cannot replace an existing password.</p>}
 </Form>}
 {screen!=="login"&&<p><Link href="/login">Return to sign in</Link></p>}
 </>;
}
