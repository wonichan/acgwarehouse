<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ApiError, getMonthlyCheckIns } from '@/api/client'
import type { MonthlyCheckInsResponse } from '@/api/client'

const WEEKDAY_LABELS = ['一', '二', '三', '四', '五', '六', '日'] as const

const loading = ref(false)
const error = ref<string | null>(null)
const checkInDates = ref<Set<string>>(new Set())
const totalPoints = ref(0)

function getCSTToday(): { year: number; month: number; day: number; dateStr: string } {
  const formatter = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  })
  const parts = formatter.formatToParts(new Date())
  const year = Number(parts.find((p) => p.type === 'year')?.value)
  const month = Number(parts.find((p) => p.type === 'month')?.value)
  const day = Number(parts.find((p) => p.type === 'day')?.value)
  return {
    year,
    month,
    day,
    dateStr: `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`,
  }
}

const today = getCSTToday()

const viewYear = ref(today.year)
const viewMonth = ref(today.month)

const monthLabel = computed(() => `${viewYear.value} 年 ${viewMonth.value} 月`)

const canGoNext = computed(() => {
  const nextMonth = viewMonth.value === 12 ? 1 : viewMonth.value + 1
  const nextYear = viewMonth.value === 12 ? viewYear.value + 1 : viewYear.value
  return !(nextYear > today.year || (nextYear === today.year && nextMonth > today.month))
})

interface CalendarCell {
  day: number
  dateStr: string
  isCurrentMonth: boolean
  isCheckedIn: boolean
  isToday: boolean
}

const calendarCells = computed<CalendarCell[]>(() => {
  const cells: CalendarCell[] = []
  const year = viewYear.value
  const month = viewMonth.value
  const firstDayMondayIndex = (new Date(year, month - 1, 1).getDay() + 6) % 7
  const daysInMonth = new Date(year, month, 0).getDate()
  for (let i = 0; i < firstDayMondayIndex; i++) {
    cells.push({ day: 0, dateStr: '', isCurrentMonth: false, isCheckedIn: false, isToday: false })
  }
  for (let day = 1; day <= daysInMonth; day++) {
    const dateStr = `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`
    cells.push({
      day,
      dateStr,
      isCurrentMonth: true,
      isCheckedIn: checkInDates.value.has(dateStr),
      isToday: dateStr === today.dateStr,
    })
  }
  const trailing = (7 - (cells.length % 7)) % 7
  for (let i = 0; i < trailing; i++) {
    cells.push({ day: 0, dateStr: '', isCurrentMonth: false, isCheckedIn: false, isToday: false })
  }
  return cells
})

const todayCheckedIn = computed(() => checkInDates.value.has(today.dateStr))

async function loadMonth(): Promise<void> {
  loading.value = true
  error.value = null
  try {
    const data: MonthlyCheckInsResponse = await getMonthlyCheckIns(viewYear.value, viewMonth.value)
    checkInDates.value = new Set(data.dates)
    totalPoints.value = data.total_points
  } catch (e) {
    checkInDates.value = new Set()
    totalPoints.value = 0
    error.value = e instanceof ApiError ? e.message : '签到记录加载失败'
  } finally {
    loading.value = false
  }
}

function goPrevMonth(): void {
  if (viewMonth.value === 1) {
    viewMonth.value = 12
    viewYear.value -= 1
  } else {
    viewMonth.value -= 1
  }
}

function goNextMonth(): void {
  if (!canGoNext.value) return
  if (viewMonth.value === 12) {
    viewMonth.value = 1
    viewYear.value += 1
  } else {
    viewMonth.value += 1
  }
}

watch([viewYear, viewMonth], () => {
  loadMonth()
})

onMounted(() => {
  loadMonth()
})
</script>

<template>
  <div class="panel checkin-panel">
    <div class="panel-head">
      <div>
        <p class="eyebrow">签到日历</p>
        <h3>每日签到记录</h3>
      </div>
      <span class="tag is-hot">累计积分 {{ totalPoints }}</span>
    </div>

    <div class="checkin-nav">
      <button class="btn btn-secondary btn-small" type="button" :disabled="loading" @click="goPrevMonth">‹ 上一月</button>
      <span class="checkin-month-label" aria-live="polite">{{ monthLabel }}</span>
      <button class="btn btn-secondary btn-small" type="button" :disabled="loading || !canGoNext" @click="goNextMonth">下一月 ›</button>
    </div>

    <div v-if="loading" class="checkin-loading" role="status" aria-label="签到记录加载中">
      <span class="sr-only">正在加载签到记录</span>
      <div class="checkin-grid">
        <span v-for="i in 35" :key="i" class="checkin-cell checkin-cell--skeleton"></span>
      </div>
    </div>

    <div v-else-if="error" class="checkin-error" role="alert">
      <p class="checkin-error__msg">{{ error }}</p>
      <button class="btn btn-secondary btn-small" type="button" @click="loadMonth">重试</button>
    </div>

    <div v-else class="checkin-calendar" role="grid" :aria-label="`${monthLabel} 签到日历`">
      <div class="checkin-weekdays" role="row">
        <span v-for="label in WEEKDAY_LABELS" :key="label" class="checkin-weekday" role="columnheader">{{ label }}</span>
      </div>
      <div class="checkin-grid">
        <div
          v-for="(cell, index) in calendarCells"
          :key="index"
          class="checkin-cell"
          :class="{
            'checkin-cell--blank': !cell.isCurrentMonth,
            'checkin-cell--checked': cell.isCheckedIn,
            'checkin-cell--today': cell.isToday,
          }"
          role="gridcell"
          :aria-label="cell.isCurrentMonth ? `${cell.day} 日${cell.isCheckedIn ? '，已签到' : ''}${cell.isToday ? '，今日' : ''}` : undefined"
        >
          <template v-if="cell.isCurrentMonth">
            <span class="checkin-day" aria-hidden="true">{{ cell.day }}</span>
            <span v-if="cell.isCheckedIn" class="sr-only">已签到</span>
            <span v-if="cell.isToday && cell.isCheckedIn" class="checkin-today-badge">今日已签到</span>
          </template>
        </div>
      </div>
    </div>

    <p v-if="!loading && !error && todayCheckedIn" class="checkin-hint meta">今日已完成签到，明日再来获取积分。</p>
    <p v-else-if="!loading && !error" class="checkin-hint meta">每日首次访问个人中心自动签到，每次获得 10 积分。</p>
  </div>
</template>

<style scoped>
.checkin-panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.checkin-nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.checkin-month-label {
  font-family: var(--font-mono);
  font-size: var(--text-sm);
  font-weight: 700;
  color: var(--fg);
}

.checkin-weekdays {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: var(--space-1);
  margin-bottom: var(--space-2);
}

.checkin-weekday {
  text-align: center;
  font-size: var(--text-xs);
  font-weight: 700;
  color: var(--muted);
  padding: var(--space-1) 0;
}

.checkin-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: var(--space-1);
}

.checkin-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  aspect-ratio: 1;
  min-height: 36px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-soft);
  background: var(--surface);
  transition: background var(--motion-fast) ease, border-color var(--motion-fast) ease;
}

.checkin-cell--blank {
  border-color: transparent;
  background: transparent;
}

.checkin-day {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--fg-2);
  line-height: 1;
}

.checkin-cell--checked {
  background: var(--accent);
  border-color: var(--accent);
}

.checkin-cell--checked .checkin-day {
  color: var(--accent-on);
  font-weight: 700;
}

.checkin-cell--today {
  border: 2px solid var(--accent);
  box-shadow: 0 0 0 2px var(--surface) inset;
}

.checkin-cell--today.checkin-cell--checked {
  border-color: var(--accent-active);
}

.checkin-today-badge {
  font-size: 9px;
  font-weight: 700;
  color: var(--accent-on);
  line-height: 1;
  margin-top: var(--space-1);
  letter-spacing: 0.02em;
}

.checkin-cell--skeleton {
  background: var(--surface-warm);
  border-color: var(--border-soft);
  animation: checkin-pulse 1.4s ease-in-out infinite;
}

@keyframes checkin-pulse {
  0%, 100% { opacity: 0.5; }
  50% { opacity: 0.9; }
}

@media (prefers-reduced-motion: reduce) {
  .checkin-cell--skeleton {
    animation: none;
    opacity: 0.7;
  }
  .checkin-cell {
    transition: none;
  }
}

.checkin-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-6) 0;
  text-align: center;
}

.checkin-error__msg {
  color: var(--muted);
  font-size: var(--text-sm);
  margin: 0;
}

.checkin-hint {
  margin: 0;
  text-align: center;
}

@media (max-width: 480px) {
  .checkin-cell {
    min-height: 32px;
  }
  .checkin-day {
    font-size: var(--text-xs);
  }
  .checkin-today-badge {
    font-size: 8px;
  }
  .checkin-nav {
    flex-wrap: wrap;
  }
}
</style>
