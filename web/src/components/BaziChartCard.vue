<template>
  <n-card title="🔮 八字命盘" size="small" class="card">
    <!-- 四柱 -->
    <div class="pillars">
      <div v-for="p in pillars" :key="p.name" class="pillar">
        <div class="p-name">{{ p.name }}</div>
        <div class="p-stem-branch">{{ p.stem }}{{ p.branch }}</div>
        <div class="p-shishen">{{ p.shiShen }}</div>
        <div class="p-nayin">{{ p.naYin }}</div>
      </div>
    </div>

    <n-divider/>

    <!-- 基本信息 -->
    <div class="info-row"><span class="label">日主</span><strong>{{ dayGan }}</strong><span class="dim">（{{ dayGanWuxing }}）</span></div>
    <div class="info-row"><span class="label">农历</span><span>{{ lunarDate }}</span></div>

    <n-divider/>

    <!-- 四柱详情表 -->
    <div class="detail-table">
      <div class="dt-header">
        <span class="dt-col">柱</span><span class="dt-col">天干</span><span class="dt-col">地支</span>
        <span class="dt-col">十神</span><span class="dt-col">纳音</span><span class="dt-col">空亡</span>
        <span class="dt-col">地势</span><span class="dt-col">旬</span>
      </div>
      <div v-for="p in pillars" :key="'dt-'+p.name" class="dt-row">
        <span class="dt-col">{{ p.name }}</span>
        <span class="dt-col fw">{{ p.stem }}</span>
        <span class="dt-col fw">{{ p.branch }}</span>
        <span class="dt-col shishen">{{ p.shiShen }}</span>
        <span class="dt-col dim">{{ p.naYin }}</span>
        <span class="dt-col dim">{{ p.xunKong }}</span>
        <span class="dt-col dim">{{ p.diShi }}</span>
        <span class="dt-col dim">{{ p.xun }}</span>
      </div>
    </div>

    <!-- 藏干 -->
    <div class="hidegan-row">
      <span class="label">藏干：</span>
      <span v-for="p in pillars" :key="'hg-'+p.name" class="hg-item">
        {{ p.name }} {{ (p.hideGan||[]).join(' ') }}
      </span>
    </div>

    <n-divider/>

    <!-- 命宫/身宫/胎元 -->
    <div class="extra-row">
      <span>命宫：<strong>{{ mingGong }}</strong><span class="dim">（{{ mingGongNaYin }}）</span></span>
      <span>身宫：<strong>{{ shenGong }}</strong><span class="dim">（{{ shenGongNaYin }}）</span></span>
      <span>胎元：<strong>{{ taiYuan }}</strong><span class="dim">（{{ taiYuanNaYin }}）</span></span>
    </div>

    <n-divider/>

    <!-- 五行 -->
    <div class="wuxing">
      <div v-for="(v,k) in wuxing" :key="k" class="wx-row">
        <span class="wx-label">{{ k }}</span>
        <n-progress :percentage="v/8*100" :height="12" :color="wxColors[k]" :show-indicator="false"/>
        <span class="wx-count">{{ v }}</span>
      </div>
    </div>

    <n-divider/>

    <!-- 大运 -->
    <div class="section-title">大运</div>
    <div class="dayun">
      <n-tag v-for="(d,i) in dayun" :key="i" size="small" class="dy-tag"
        :type="i===currentDayunIdx?'warning':'default'">
        {{ d.startAge }}-{{ d.endAge }}岁 {{ d.ganZhi }}
      </n-tag>
    </div>
  </n-card>
</template>
<script setup lang="ts">
import {computed} from 'vue'
import {NCard,NDivider,NProgress,NTag} from 'naive-ui'
const props=defineProps<{data:any}>()
const pillars=computed(()=>props.data?.pillars||[])
const dayGan=computed(()=>props.data?.dayGan||'')
const dayGanWuxing=computed(()=>props.data?.dayGanWuxing||'')
const lunarDate=computed(()=>props.data?.lunarDate||'')
const wuxing=computed(()=>props.data?.wuxing||{})
const dayun=computed(()=>props.data?.dayun||[])
const mingGong=computed(()=>props.data?.mingGong||'')
const mingGongNaYin=computed(()=>props.data?.mingGongNaYin||'')
const shenGong=computed(()=>props.data?.shenGong||'')
const shenGongNaYin=computed(()=>props.data?.shenGongNaYin||'')
const taiYuan=computed(()=>props.data?.taiYuan||'')
const taiYuanNaYin=computed(()=>props.data?.taiYuanNaYin||'')
const wxColors:Record<string,string>={木:'#5B8C5A',火:'#C44B3C',土:'#D4A853',金:'#E8E1D5',水:'#3C5A7D'}
const currentDayunIdx=computed(()=>{
  // Find which dayun the user is currently in (based on a rough age calculation from birthday)
  if (!props.data?.birthday) return -1
  const birthYear=parseInt(props.data.birthday)||0
  const age=new Date().getFullYear()-birthYear
  return dayun.value.findIndex((d:any)=>age>=d.startAge&&age<=d.endAge)
})
</script>
<style scoped>
.card{margin:8px 0;text-align:left}
.pillars{display:flex;justify-content:space-around;text-align:center;margin-bottom:4px}
.pillar{flex:1}
.p-name{font-size:12px;color:var(--n-text-color-3)}
.p-stem-branch{font-size:26px;font-weight:bold;line-height:1.3}
.p-shishen{font-size:13px;color:#D4A853;margin-top:1px}
.p-nayin{font-size:11px;color:var(--n-text-color-3)}
.info-row{margin:4px 0;font-size:13px}
.info-row .label{color:var(--n-text-color-3);margin-right:8px}
.dim{color:var(--n-text-color-3);font-size:12px}
.fw{font-weight:bold}
.shishen{color:#D4A853}
.detail-table{font-size:12px;margin:4px 0}
.dt-header,.dt-row{display:flex;justify-content:space-between;padding:2px 0}
.dt-header{border-bottom:1px solid var(--n-border-color);color:var(--n-text-color-3);margin-bottom:2px}
.dt-col{flex:1;text-align:center;min-width:0}
.dt-row{border-bottom:1px solid rgba(255,255,255,0.04)}
.hidegan-row{font-size:12px;color:var(--n-text-color-3);margin:6px 0}
.hg-item{margin-right:12px}
.extra-row{display:flex;gap:16px;font-size:13px;flex-wrap:wrap}
.extra-row span{white-space:nowrap}
.section-title{font-size:13px;color:var(--n-text-color-3);margin-bottom:6px}
.wx-row{display:flex;align-items:center;gap:8px;margin:4px 0}
.wx-label{width:20px}.wx-count{width:20px;text-align:right}
.dy-tag{margin:2px}
</style>
