"use client";
import {type FormEvent,type ReactNode} from "react";
export type Grant={role_id:string;scope:"ORG"|"TEAM"|"SELF";team_id:string|null};
export type Row={id:string;grants?:Grant[];name?:string;display_name?:string;canonical_email?:string;status?:string;state?:string;delivery_state?:string;revision:number;permissions?:string[];current?:boolean;created_at?:string;expires_at?:string;action?:string;resource_id?:string;member_id?:string;archived_at?:string};
export type User={id:string;user_id:string;name:string;email:string;verified:boolean;mfa_enabled:boolean;organization_id:string|null};
export type Organization={id:string;name:string;permissions:string[];scoped_teams:{id:string;permissions:string[]}[];member_id:string};
export type Act=(fn:()=>Promise<void>,success?:string)=>Promise<void>;
export const text=(data:FormData,key:string)=>String(data.get(key)??"");
export function Field({name,label,type="text",required=true,autoComplete,minLength}:{name:string;label:string;type?:string;required?:boolean;autoComplete?:string;minLength?:number}){
 return <label>{label}<input name={name} type={type} required={required} autoComplete={autoComplete} minLength={minLength} maxLength={type==="password"?256:254}/></label>;
}
export function Form({children,label,submit,busy}:{children:ReactNode;label:string;submit:(data:FormData)=>Promise<void>;busy:boolean}){
 return <form onSubmit={(e:FormEvent<HTMLFormElement>)=>{e.preventDefault();void submit(new FormData(e.currentTarget));}}>{children}<button disabled={busy} type="submit">{busy?"Working...":label}</button></form>;
}
export function confirmAction(target:string,consequence:string){return window.confirm(target+"\n\n"+consequence);}
