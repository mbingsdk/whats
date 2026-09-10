export class IdentityError extends Error {
 code:string;status:number;
 constructor(code:string,status:number){super(code.replaceAll("_"," ").toLowerCase());this.code=code;this.status=status;}
}
export async function request<T>(path:string,method="GET",body?:unknown,revision?:number):Promise<T>{
 if(!path.startsWith("/api/v1/")||path.includes("://"))throw new Error("Invalid API path");
 const headers:Record<string,string>={};
 if(method!=="GET"){
  const csrf=await fetch("/api/v1/auth/csrf",{credentials:"same-origin",cache:"no-store"});
  if(!csrf.ok)throw new IdentityError("SERVICE_UNAVAILABLE",csrf.status);
  headers["X-CSRF-Token"]=(await csrf.json()).csrf_token;headers["Content-Type"]="application/json";
  if(revision)headers["If-Match"]=`"${revision}"`;
 }
 const response=await fetch(path,{method,headers,credentials:"same-origin",cache:"no-store",body:body===undefined?undefined:JSON.stringify(body)});
 const result=await response.json();
 if(!response.ok)throw new IdentityError(result.error?.code??"REQUEST_FAILED",response.status);
 return result as T;
}

export type Page<T>={items:T[];has_more:boolean;next_cursor:string|null};
export async function allPages<T>(path:string):Promise<T[]>{
 const items:T[]=[];let cursor:string|null=null;
 do {const page:Page<T>=await request(path+(cursor?"?cursor="+encodeURIComponent(cursor):""));items.push(...page.items);cursor=page.next_cursor;}while(cursor);
 return items;
}
