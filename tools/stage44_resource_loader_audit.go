// Read-only audit of the 16 INI inputs in server/skill_catalog.go.
// Reports aggregate counts and hashes; never prints game lines or identifiers.
package main

import (
 "bufio"
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "flag"
 "fmt"
 "io"
 "os"
 "path/filepath"
 "strconv"
 "strings"
)

type skill struct{ script, static string }
type staticInfo struct{ fields int; min, max int32 }
type fileStat struct {
 Name string `json:"name"`
 Bytes int64 `json:"bytes"`
 SHA256 string `json:"sha256"`
 Sections int `json:"sections"`
 DuplicateHeaders int `json:"duplicate_headers"`
 Fields int `json:"fields"`
 NULBytes int `json:"nul_bytes"`
 MaxLineBytes int `json:"max_line_bytes"`
}
type report struct {
 Files []fileStat `json:"files"`
 InputsRead int `json:"inputs_read"`
 Indexed int `json:"indexed_supported_scripts"`
 LevelOnePreflight int `json:"level_one_preflight"`
 ReasonCounts map[string]int `json:"reason_counts"`
 MergedActionSections int `json:"merged_action_sections"`
 ShopLegacyIdentical *bool `json:"shop_legacy_identical,omitempty"`
}
func asInt(v string) int32 { n,_:=strconv.ParseInt(strings.TrimSpace(v),10,32); return int32(n) }
func digest(path string)(string,int64,int,error){
 f,err:=os.Open(path);if err!=nil{return "",0,0,err};defer f.Close()
 h:=sha256.New();buf:=make([]byte,65536);var size int64;nuls:=0
 for {n,e:=f.Read(buf);if n>0{h.Write(buf[:n]);size+=int64(n);for _,v:=range buf[:n]{if v==0{nuls++}}};if e==io.EOF{break};if e!=nil{return "",0,0,e}}
 return hex.EncodeToString(h.Sum(nil)),size,nuls,nil
}
func scan(path,name string,onSection func(string),onField func(string,string,string))(fileStat,error){
 out:=fileStat{Name:name};var err error
 out.SHA256,out.Bytes,out.NULBytes,err=digest(path);if err!=nil{return out,err}
 f,err:=os.Open(path);if err!=nil{return out,err};defer f.Close()
 scanner:=bufio.NewScanner(f);scanner.Buffer(make([]byte,64*1024),4*1024*1024)
 section:="";seen:=map[string]bool{}
 for scanner.Scan(){
  line:=scanner.Text();if len(line)>out.MaxLineBytes{out.MaxLineBytes=len(line)}
  if strings.HasPrefix(line,"\ufeff"){line=line[len("\ufeff"):]};line=strings.TrimSpace(line)
  if line==""||line[0]==';'||line[0]=='#'{continue}
  if line[0]=='['&&line[len(line)-1]==']'{section=strings.TrimSpace(line[1:len(line)-1]);out.Sections++;if seen[section]{out.DuplicateHeaders++};seen[section]=true;onSection(section);continue}
  if section==""{continue};key,val,ok:=strings.Cut(line,"=");if !ok{continue}
  out.Fields++;onField(section,strings.TrimSpace(key),strings.TrimSpace(val))
 }
 if err:=scanner.Err();err!=nil{return out,fmt.Errorf("scan %s: %w",name,err)}
 return out,nil
}
func levelOne(st *staticInfo,props map[string]int32)bool{
 if st==nil||st.fields==0{return false}
 for n:=st.min;n<=st.max&&n>0;n++{if props[strconv.Itoa(int(n))]==1{return true};if n==2147483647{break}}
 return false
}
func audit(skillNew,skillRoot,actionRoot,shop,oldShop string)(report,error){
 out:=report{ReasonCounts:map[string]int{}}
 skillFiles:=[]string{"skill_new.ini","skill_static.ini","skill_normal_varprop.ini","skill_lock_varprop.ini","skill_consume.ini","damage_calculate.ini","attack_hitshape.ini","attack_targetshape.ini"}
 actionFiles:=[]string{"zhaoshi_player.ini","zhaoshi_player_2.ini","zhaoshi_player_dodge.ini","zhaoshi_player_parry.ini","zhaoshi_clone.ini"}
 buffFiles:=[]string{"buff_new.ini","buff_static.ini","buff_varprop.ini"}
 inputs:=append(append(skillFiles,actionFiles...),buffFiles...)
 skills:=map[string]*skill{};statics:=map[string]*staticInfo{};normal:=map[string]int32{};lock:=map[string]int32{};actions:=map[string]bool{}
 for _,name:=range inputs{
  path:=filepath.Join(skillRoot,name);if name=="skill_new.ini"{path=skillNew}
  isAction:=false;for _,a:=range actionFiles{if a==name{isAction=true;break}};if isAction{path=filepath.Join(actionRoot,name)}
  localActions:=map[string]bool{}
  onSection:=func(sec string){switch name{
   case "skill_new.ini":if _,ok:=skills[sec];!ok{skills[sec]=&skill{}}
   case "skill_static.ini":if _,ok:=statics[sec];!ok{statics[sec]=&staticInfo{}}
   default:if isAction{if _,ok:=localActions[sec];!ok{localActions[sec]=false}}
  }}
  onField:=func(sec,key,val string){lower:=strings.ToLower(key);switch name{
   case "skill_new.ini":if lower=="script"{skills[sec].script=val}else if lower=="staticdata"{skills[sec].static=val}
   case "skill_static.ini":st:=statics[sec];st.fields++;if lower=="minvarpropno"{st.min=asInt(val)}else if lower=="maxvarpropno"{st.max=asInt(val)}
   case "skill_normal_varprop.ini":if lower=="level"{normal[sec]=asInt(val)}
   case "skill_lock_varprop.ini":if lower=="level"{lock[sec]=asInt(val)}
   default:if !isAction||strings.HasPrefix(key,"trigger_"){return};if _,err:=strconv.Atoi(key);err!=nil{return};p:=strings.Split(val,";");if len(p)>=5&&strings.TrimSpace(p[0])!=""{localActions[sec]=true}
  }}
  stat,err:=scan(path,name,onSection,onField);if err!=nil{return out,err};out.Files=append(out.Files,stat);out.InputsRead++
  if isAction{for sec,valid:=range localActions{if name=="zhaoshi_clone.ini"{if _,exists:=actions[sec];exists{continue}};actions[sec]=valid}}
 }
 out.MergedActionSections=len(actions)
 for id,s:=range skills{
  if !strings.EqualFold(s.script,"SkillNormal")&&!strings.EqualFold(s.script,"SkillLock"){out.ReasonCounts["unsupported_script"]++;continue}
  out.Indexed++;sid:=asInt(s.static);if sid<=0{out.ReasonCounts["missing_static_id"]++;continue}
  st:=statics[strconv.Itoa(int(sid))];if st==nil||st.fields==0{out.ReasonCounts["missing_static"]++;continue}
  props:=normal;if strings.EqualFold(s.script,"SkillLock"){props=lock};if !levelOne(st,props){out.ReasonCounts["missing_level_varprop"]++;continue}
  if !actions[id]{out.ReasonCounts["missing_player_action"]++;continue};out.LevelOnePreflight++
 }
 if shop!=""||oldShop!=""{if shop==""||oldShop==""{return out,fmt.Errorf("both --shop and --legacy-shop are required")};a,_,_,err:=digest(shop);if err!=nil{return out,err};b,_,_,err:=digest(oldShop);if err!=nil{return out,err};same:=a==b;out.ShopLegacyIdentical=&same}
 return out,nil
}
func main(){
 skillNew:=flag.String("skill-new","","selected private skill_new.ini")
 skillRoot:=flag.String("skill-root","","directory containing the other skill and buff INIs")
 actionRoot:=flag.String("action-root","","directory containing the five action INIs")
 shop:=flag.String("shop","","optional selected shop.ini")
 legacy:=flag.String("legacy-shop","","optional legacy shop.ini")
 flag.Parse();if *skillNew==""||*skillRoot==""||*actionRoot==""{fmt.Fprintln(os.Stderr,"required: --skill-new --skill-root --action-root");os.Exit(2)}
 result,err:=audit(*skillNew,*skillRoot,*actionRoot,*shop,*legacy);if err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}
 if err:=json.NewEncoder(os.Stdout).Encode(result);err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}
}
