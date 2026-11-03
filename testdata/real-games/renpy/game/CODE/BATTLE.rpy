#при вызове боя можно параметры задавать
#$ renpy.call_in_new_context("battle", bg="bg battle forest", allow_attack=[ 0 ], allow_action=[ 0, 3 ])
#bg - фон
#allow_attack - номера доступных атак [ 0 ] - это самая первая, одиночный удар
#allow_action - доступные действия [ 0, 3 ] - это атака и пропуск хода
#если ничего не указывать, то будет полный фарш, без ограничений
#$ renpy.call_in_new_context("battle")
#но фон по умолчанию - подвал


# центр расположения команд
define pers_cx = .2
define pers_cy = .8

# радиус распределения для 1, 2, 3, 4 персонажей в группе
define pers_rx = ( 0, 150, 200, 250 )
define pers_ry = ( 0, 100, 125, 150 )

# очков на ход по умолчанию
define b_ap_default = 5

# максимум очков на ход
define b_ap_max = 8

# сколько ОД добавляется в начале каждого нового хода
define b_ap_restore = 2

# уголки для наведения на врага
image b_corner1:
    xysize (40, 40)

    contains:
        "btn corner"
        subpixel True
        align (0, 0)
        ease .5 align (1., 1.)
        ease .5 align (0, 0)
        repeat
image b_corner2 = Transform("b_corner1", xzoom=-1.)
image b_corner3 = Transform("b_corner1", yzoom=-1.)
image b_corner4 = Transform("b_corner3", xzoom=-1.)

# стили для информационного окна
style b_frame is frame:
    padding (48, 48)
    xsize 480
    background "#000c"
    align (.5, .5)

style b_vbox is vbox:
    xfill True
    align (.5, .5)

style b_hbox is hbox:
    xpadding 20

style b_label is label:
    xalign .5

style b_text is text

# кнопка-ромбик (выход, инфо)
style b_imagebutton is button:
    xysize (100, 100)
    anchor (.5, .5)
    background Frame("btn idle", 0, 0)
    hover_background Frame("btn hover", 0, 0)

style b_imagebutton_text is button_text:
    align (.5, .5)
    text_align .5
    layout "subtitle"

# кнопка в списке
style b_button is button:
    xalign .5
    xysize (535, 105)
    background Frame("button2 idle", 6, 6)
    hover_background Frame("button2 hover", 6, 6)
    insensitive_background Frame("button2 insensitive", 6, 6)
    selected_background Frame("button2 selected", 6, 6)
    selected_hover_background Frame("button2 selected_hover", 6, 6)

style b_button_text is button_text:
    align (.5, .5)
    text_align .5
    layout "subtitle"
    bold True

# плавно показать/скрыть
transform b_show_hide(t=.25):
    alpha 0

    on show:
        ease t alpha 1

    on hide:
        ease t alpha 0

# отзеркаливание
transform pers_xzoom(xzoom=-1):
    xzoom xzoom

transform b_no_choose(z=.75, b=.2):
    zoom z
    # matrixcolor BrightnessMatrix(b)

# части анимаций отвечающие за атаки
# чтобы добавить реакцию противника на удары и вычислить силу ударов
define b_attack_parts = { "hit": 10, "hit2": 20, "hit3": 30, "attack": 5 }

# анимации без перемещений, когда показан хелсбар и доступно наведение
define b_stay_parts = [ "stay", "ouch", "dead" ]

# все последовательности анимаций
define b_act_parts = {
    "hit": [ "start", .35, "hit", .35, "return" ],
    "hit2": [ "start", .35, "hit2", .35, "return" ],
    "hit3": [ "start", .35, "hit3", .35, "return" ],
    "ouch": [ "ouch" ],
    "skip": [ .75 ],
    "item": [ .05 ]
}

# звуки для указанных частей анимации
define b_sounds = {
    "start": "b_start",
    "hit": "b_hit",
    "return": "b_return"
}

# размеры бара
define b_bar_xysize = (222, 12)

# ниже какого значения бар меняет цвет
define b_k_min = .33

# белая полоска
image b_bar_w = Transform("#fff", xysize=b_bar_xysize)

# белая полоска для рамки
image b_bar_w0 = Transform("#fffd", xysize=(b_bar_xysize[0] + 2, b_bar_xysize[1] + 2))

# перекрашиваем и подрезаем полоску
transform b_paint(color="#bbb", k=1.):
    matrixcolor TintMatrix(color)
    crop(0, 0, k, 1.)

# для обозначения хитов, отлетающих вверх
transform b_dec_at(x, y, t=1, dy=444):
    subpixel True

    parallel:
        alpha 0 zoom 0
        linear t * .15 alpha 1 zoom 1
        t * .15
        linear t * .7 alpha 0 zoom 0

    parallel:
        anchor (.5, 1.) pos (x, y)
        easeout t pos (x, y - dy)

# вспышка при получении урона
transform b_ouch_at(b=1, w=.1, t=.25):
    matrixcolor BrightnessMatrix(0)
    w
    ease t * .5 matrixcolor BrightnessMatrix(b)
    ease t * .5 matrixcolor BrightnessMatrix(0)

init python:
    def b_is_hover(x1=1, x0=0):
        return x1 if b_state in [ "choose_who", "choose_target" ] else x0

    # картинка прицела из движущихся уголков
    def b_corners(w, h):
        return Composite( (w, h), (0, 0), "b_corner1", (w - 40, 0), "b_corner2", (0, h - 40), "b_corner3", (w - 40, h - 40), "b_corner4" )

# реакция на курсор
transform b_hover_at(t=.2, b=.1):
    matrixcolor BrightnessMatrix(0)

    on idle:
        ease t matrixcolor BrightnessMatrix(0)

    on hover, selected:
        ease t matrixcolor BrightnessMatrix(b)

# чтобы цифры могли вылетать из нескольких персонажей
default b_dec_tag_index = 0

# уникальный номер серии ударов, чтобы старые таймеры не срабатывали после смены анимации
default b_attack_event_index = 0

# номер текущего хода, нужен чтобы манекен не лечился дважды за один ход
default b_turn_number = 0

# защита от повторного лечения манекена внутри одного и того же хода
default b_dummy_heal_key = None

# манекен уже сделал свой ход в этом цикле
default b_dummy_turn_done = False

style b_alias is text:
    size 20
    color "#fff"
    outlines [ (1, "#000d", 0, 0) ]
    text_align.5
    layout "subtitle"

transform b_corners_pos(x, y):
    anchor (.5, 1.)
    pos (x, y)

transform b_xoffset(x=200):
    xoffset x

init python:
    # для авто-tag
    pers_tag_index = 1
    pers_tag_name = "warrior%s"

    # пока у всех спрайты селены и чучела
    pers_sprites = { "hero": "selena", "friend": "selena", "enemy": "dummy", "boss": "dummy" }

    # положение лиц на спрайте, чтобы отобразить в углу экрана
    pers_faces = { "selena": (110, 250, 120, 120), "dummy": (100, 233, 125, 125), "bandit": (130, 235, 120, 120) }

    # полоска, заполненная на k [0..1]
    # def b_bar(k=1, color_left="#084", color_min="#dc4", color_right="#000", k_min=b_k_min, z=0):
        # color_min = color_min or color_left
        # color = (color_min if k <= k_min else color_left)
        # left = At("b_bar_w", b_paint(color, k * 1.))
        # right = At("b_bar_w", b_paint(color_right, 1.))
        # return Transform(Composite(b_bar_xysize, (0, 0), "b_bar_w0", (1, 1), right, (1, 1), left))

    # полоска, заполненная на k [0..1]
    def b_bar(k=1):
        return Transform(Composite(b_bar_xysize, (0, 0), "btn hpbg", (4, 0), Transform("btn hp", crop=(0, 0, k, 1.))))

    # статичная анимация?
    def b_stay(act):
        return act in b_stay_parts

    # класс для любого персонажа
    # числовые параметры hp, strength, defense, speed - это здоровье, сила, защита и скорость
    # к целочисленным параметрам в пару создаются прокачиваемые потолки: hp_max, strength_max и т.д.
    # kind - тип персонажа (hero, friend, enemy, boss)
    class Pers(object):
        def __init__(self, alias=_("Противник"), kind="enemy", hp=100, strength=50, defense=10, speed=50, hunger=0, moral=0, lore=0, spr=None, h=anima_char_h, hit_acts=None, turn_end=[ ], **kwargs):
            # Если спрайт не передали явно, берём его по типу персонажа.
            if not spr:
                spr = pers_sprites.get(kind, None)

            # Если kind нестандартный, например "bandit",
            # считаем его врагом и берём обычный enemy-спрайт.
            # Раньше spr мог стать None, и бой падал/ломался на self.spr.show().
            if not spr:
                spr = pers_sprites.get("enemy" if b_is_enemy(kind) else "hero", None)

            self.spr_def = spr if spr else "null"

            tag = kwargs.get("tag", None)

            # авто-tag
            if tag is None:
                tag = pers_tag_name % (pers_tag_index)
                store.pers_tag_index += 1

            # тег для вывода спрайта, он же уникальный id
            self.tag = tag
            # отображаемое имя персонажа
            self.alias = alias
            # тип персонажа (герой, друг, враг, босс)
            self.kind = kind
            # высота персонажа (для хелсбара и прочего над головой)
            self.h = h

            # очерёдность вывода на экран
            self.zorder = None
            # текущие Очки Действий
            self.ap = b_ap_default
            # дополнительные ОД с прошлого хода
            self.ap_add = 0

            # основные параметры
            kwargs["hp"] = hp
            kwargs["strength"] = strength
            kwargs["defense"] = defense
            kwargs["speed"] = speed
            kwargs["hunger"] = hunger
            kwargs["moral"] = moral
            kwargs["lore"] = lore

            self.spr = (TAnimaSpr(spr, h=h, enemy=b_is_enemy(kind)) if spr else None)

            # потолок для цифровых параметров
            for i in [ i for i in kwargs.keys() if not str(i).endswith("_max") ]:
                if isinstance(kwargs[i], (int, float)):
                    kwargs[i + "_max"] = kwargs.get(i + "_max", kwargs[i])

            self.d = kwargs
            self.act_i = None

            self.x, self.y = None, None

            self.act_acts = b_act_acts_default

            if hit_acts is None:
                hit_acts = ( b_enemy_hit_acts_default if b_is_enemy(self.kind) else b_hero_hit_acts_default )

            self.hit_acts = hit_acts[:]

            self.hit_act = None

            self.turn_end = turn_end

        # сектор для circle_bar
        def sector(self):
            # *** заменить на вычисление с учётом параметров персонажа
            d = 45
            a1 = rndf(90, 360 - d)
            a2 = a1 + d

            return a1, a2

        # выполнить скрипт в конце хода
        def turn_exec(self, all=None):
            all = all or self.turn_end

            for i in all: exec(i)

        def reset(self):
            if self.spr: self.spr.reset()

        def restart(self):
            self.reset()

        def setpause(self, on=None):
            if self.spr: self.spr.setpause(on)

        def pause(self):
            if self.spr: self.pause()

        def unpause(self):
            if self.spr: self.unpause()

        def set_act(self, act=None, restart=True):
            if self.spr and not isinstance(act, (int, float)): self.spr.set_act(act, restart)

        # получить спрайт, который можно показать на экране
        def sprite(self, act=None, restart=None):
            if self.spr: return self.spr.sprite(act, restart)

            return None

        # получить параметр
        def get_par(self, par="speed"):
            res = self.d[par]

            # *** добавить влияние артефактов
            return res

        # получить параметр в допустимых пределах
        def get_k(self, par="hp"):
            return self.d[par] / self.d[par + "_max"]

        # увеличить параметр в допустимых пределах
        def inc(self, par="hp", plus=1):
            self.d[par] += plus
            self.d[par] = max(self.d[par], 0)

            if self.d[par + "_max"] > 0: self.d[par] = min(self.d[par], self.d[par + "_max"])

        # уменьшить параметр в допустимых пределах
        def dec(self, par="hp", minus=1):
            self.inc(par, -minus)

        # получить параметр в виде коэффициента относительно потолка [ 0..1 ]
        def k(self, par="hp"):
            return self.d[par] / self.d.get(self.d[par + "_max"], self.d[par])

        # восстановить часть потраченных очков действий в начале нового хода
        def restore_ap(self):
            self.ap = min(self.ap + b_ap_restore, b_ap_max)

        # посчитать координаты персонажа на экране
        def get_pos(self):
            global b_heroes, b_enemies

            party = b_heroes.pers if self.kind in ( "hero", "friend" ) else b_enemies.pers

            count = len(party)

            if not self in party: return (0, 0)

            i = party.index(self)
            x, y = 1 - pers_cx, pers_cy

            if self.kind in ( "hero", "friend" ): x = pers_cx

            x *= config.screen_width
            y *= config.screen_height

            if len(party) > 1:
                a = math.radians(i * 360 / count + 24.5)
                x = x + math.cos(a) * ( pers_rx[count - 1] if count < len(pers_rx) else max(*pers_rx) )
                y = y + math.sin(a) * ( pers_ry[count - 1] if count < len(pers_ry) else max(*pers_ry) )

            return int(x), int(y)

        # длительность анимации
        def get_act_t(self, act=None):
            act = act or b_act_part.get(self.tag, None)

            if self.spr: return self.spr.get_act_t(act)

            return 0

        # текущие координаты над головой
        def get_xy(self):
            self.x, self.y = self.get_pos()
            x, y = self.x, self.y
            y -= self.h

            return x, y

        # показать стрелку над головой
        def show_arrow(self, tag=None):
            if not self.spr:
                return

            self.x, self.y = self.get_pos()
            self.spr.show_arrow(self.x, self.y, tag or self.tag, self.zorder)

        # скрыть стрелку над головой
        def hide_arrow(self, tag=None):
            if self.spr:
                self.spr.hide_arrow(tag or self.tag)

        # вывести спрайт на экран
        def show(self, act=None, at_list=[ ], repaint=False):
            act = act or b_act_part.get(self.tag, None)
            self.x, self.y = self.get_pos()
            x, y = (self.x, self.y)

            if b_target:
                if b_act_part.get(self.tag, None) in b_attack_parts.keys(): x, y = (b_target.x, b_target.y)

            # полоска здоровья над головой
            hp = self.get_k()
            z = self.zorder

            if (self == b_who) and b_target: z = b_target.zorder

            arrow = False

            if act == "ouch" and b_state == "action": at_list = at_list + [ b_ouch_at ]

            if self.d["hp"] >= 0:
                if self.spr:
                    if self.spr.tag == "bandit" and act == "hit":
                        at_list = at_list + [ b_xoffset ]

                    else:
                        at_list = at_list + [ b_xoffset(0) ]

                    self.spr.show(act, self.tag, at_list=at_list, zorder=z, x=x, y=y, hp=hp, who=self)

                else:
                    renpy.notify("Ошибка: у персонажа %s не задан spr" % self.tag)

            if repaint: renpy.restart_interaction()

        # убрать спрайт с экрана
        def hide(self, transition=False, repaint=True):
            if self.spr:
                self.spr.hide(self.tag)

            if transition or (transition is None): renpy.transition(transition)

            if repaint: renpy.restart_interaction()

        def timer(self):
            # таймер для переключения анимаций
            # длительность текущей анимации
            t = self.get_act_t()

            # если вместо анимации пауза
            if b_pause.get(self.tag, None):
                t = b_pause[self.tag]

            # если идёт анимация
            else:
                # по её окончанию переключаемся на следующую
                t = t or .0001

            if t: add_timer(t, BNextAct(self))

        # список кнопок и соответствующих атак
        def action_buttons(self):
            return self.act_acts

        # список кнопок и соответствующих действий
        def hit_buttons(self):
            return self.hit_acts

        # кончились ли атаки
        def end_attacks(self, index, current):
            is_int = isinstance(index, int)
            hit_act = self.hit_acts[index] if is_int else b_other_acts[index]
            alias, name, ap, acts = hit_act

            return current >= len(acts)

        # запускаем атаку
        def attack(self, index, current=0):
            # запускаем ряд анимаций атаки
            is_int = isinstance(index, int)
            hit_act = self.hit_acts[index] if is_int else b_other_acts[index]
            alias, name, ap, acts = hit_act
            self.hit_act = acts[current]
            store.b_state = "action"
            b_arrows()
            # self.ap -= ap
            self.start_act(self.hit_act[0])

        # сила текущего удара
        def power(self, act=None):
            res = None
            act = act or b_act_part[self.tag]

            if self.hit_act:
                # коэффициент удара
                # (у каждого персонажа может быть свой набор атак)
                act, power = self.hit_act

                # помножим на стандартную силу текущего удара
                power *= b_attack_parts.get(act, 0)

                # *** добавить действие параметров и артефактов
                return power

            return res

        # сколько отдельных ударов внутри текущей hit-анимации
        def attack_hit_count(self, act=None):
            act = act or b_act_part.get(self.tag, None)

            if act == "hit2":
                return 2

            if act == "hit3":
                return 3

            return 1

        # общий урон всей атаки
        def attack_total_damage(self, act=None):
            act = act or b_act_part.get(self.tag, None)
            return -self.power(act)

        # урон отдельного попадания с учётом результата своего QTE
        def attack_hit_damage(self, act=None, hit_i=0, hit_count=1):
            act = act or b_act_part.get(self.tag, None)

            # У врагов QTE нет — они бьют обычным уроном по старой формуле.
            if b_is_enemy(self.kind):
                total_p = self.attack_total_damage(act)
                return total_p / float(max(1, hit_count))

            # Базовый урон каждого отдельного удара.
            # hit  = 7
            # hit2 = 7 + 6
            # hit3 = 7 + 6 + 8
            base_hits = {
                "hit":  [ 7 ],
                "hit2": [ 7, 6 ],
                "hit3": [ 7, 6, 8 ]
            }

            hits = base_hits.get(act, [ 7 ])
            base_p = hits[min(hit_i, len(hits) - 1)]

            # Каждый QTE-бар отвечает только за свой конкретный удар.
            # Успех = x2, провал = обычный урон.
            success = False

            if hit_i < len(b_cancel_results):
                success = bool(b_cancel_results[hit_i])

            mult = 2 if success else 1

            return -base_p * mult

        # запустить отдельные срабатывания урона внутри одной hit-анимации
        def start_attack_hits(self, act=None):
            global b_attack_event_index

            act = act or b_act_part.get(self.tag, None)
            count = self.attack_hit_count(act)

            b_attack_event_index += 1
            token = b_attack_event_index

            # Каждый блок 1,2,3,4,5 длится 5 кадров.
            # Урон показываем на первом кадре каждого блока:
            # hit2: 0.00 и 0.25 сек; hit3: 0.00, 0.25 и 0.50 сек при anima_t=.05.
            for hit_i in range(count):
                delay = max(.001, hit_i * 5 * anima_t)
                p = self.attack_hit_damage(act, hit_i, count)
                timer_id = "battle_hit_%s_%s_%s" % (self.tag, token, hit_i)
                add_timer(delay, BApplyAttackHit(self, act, p, hit_i, count, token), 1, timer_id)

        # применить один отдельный удар из составной атаки
        def apply_attack_hit(self, act, p, hit_i=0, hit_count=1, token=None):
            global b_dec_tag_index, b_attack_event_index

            # защита от старых таймеров атаки
            if token is not None and token != b_attack_event_index:
                return

            # защита от старых таймеров, если анимация уже закончилась или сменилась
            if self.act_i is None:
                return

            if b_act_part.get(self.tag, None) != act:
                return

            if not b_target:
                return

            # если цель уже добита предыдущим ударом из двойной/тройной атаки,
            # не запускаем следующие отложенные попадания и ouch-анимацию
            if b_target.d["hp"] <= 0:
                return

            # количество хитов
            if p:
                # отнимаем хиты у противника
                b_target.inc("hp", p)
                color, s = "#fff", ""

                if p > 0: color, s = "#4f4", "+"

                if p < 0: color, s = "#f44", "-"

                # Разносим цифры чуть по X, чтобы 2/3 отдельных попадания не слипались в одну кучу.
                value = int(round(abs(p)))
                t = Text("%s%s" % (s, value), size=(66 if b_cancel else 44), color=color, bold=True)
                id = "dec_%s" % b_dec_tag_index
                x, y = b_target.x, b_target.y
                x += int((hit_i - (hit_count - 1) / 2.0) * 70)
                y -= b_target.h
                renpy.show(id, what=t, at_list=[ b_dec_at(x, y) ], zorder=111)
                b_dec_tag_index += 1

                if b_dec_tag_index > 11: b_dec_tag_index = 0

            # покажем реакцию на каждый отдельный удар
            b_target.start_act("ouch")

        # запускаем анимацию
        def start_act(self, act="ouch"):
            global b_who, b_target, b_act, b_act_part, b_last_act_part, b_state

            self.act_i = 0
            b_last_act_part[self.tag] = None
            b_state = "action"
            b_act[self.tag] = act
            b_act_part[self.tag] = b_act_parts[b_act[self.tag]][self.act_i]

            if not isinstance(b_act_part[self.tag], (int, float)):
                self.show(b_act_part[self.tag])

            if not b_stay(self.spr.act):
                renpy.hide(self.tag + "_hp")
                renpy.hide(self.tag + "_alias")

            if not isinstance(b_act_part[self.tag], (int, float)): self.show()

            renpy.restart_interaction()
            self.timer()

            if b_act_part[self.tag] in b_sounds: splay(b_sounds[b_act_part[self.tag]])

        def die(self):
            # ВАЖНО:
            # Раньше здесь проверялось только self.kind == "enemy".
            # Поэтому нестандартные враги вроде "bandit" после смерти
            # удалялись из b_heroes, а не из b_enemies.
            if b_is_enemy(self.kind):
                b_enemies.die(self)

            else:
                b_heroes.die(self)

        # смена этапа анимации
        def next_act(self):
            global b_who, b_target, b_act, b_act_part, b_last_act_part, b_state, b_pause, b_dec_tag_index

            if not self.act_i is None:
                self.act_i += 1

                # если этапы ещё есть, переключаемся на них
                if self.act_i < len(b_act_parts[b_act[self.tag]]):
                    part = b_act_parts[b_act[self.tag]][self.act_i]

                    # пауза
                    if isinstance(part, (int, float)):
                        b_pause[self.tag] = part
                        self.timer()

                    # следующая часть анимации
                    else:
                        b_pause[self.tag] = None
                        b_last_act_part[self.tag] = b_act_part[self.tag]
                        b_act_part[self.tag] = b_act_parts[b_act[self.tag]][self.act_i]
                        self.show(b_act_part[self.tag], repaint=True)

                        # если есть звук для этой части анимации
                        if b_act_part[self.tag] in b_sounds: splay(b_sounds[b_act_part[self.tag]])

                        # если атака, то запускаем отдельные попадания внутри анимации
                        if b_act_part[self.tag] in b_attack_parts.keys():
                            self.start_attack_hits(b_act_part[self.tag])
                        self.timer()

                # этапы кончились
                else:
                    b_pause[self.tag] = None
                    self.act_i = None

                    # if self.tag == b_who.tag: b_target, b_state = None, "choose_who"
                    # ***
                    if self.tag == b_who.tag: b_state = "choose_who"

                    self.hit_act = None
                    b_act[self.tag] = None
                    b_act_part[self.tag], b_last_act_part[self.tag] = None, None
                    self.spr.act = self.spr.default_act[:]
                    self.show()
                    renpy.end_interaction(True)

            b_arrows()

    # класс для команды из одного или кучки персонажей
    class Party(object):
        def __init__(self, *args, enemy=True, **kwargs):
            self.pers = list(args)
            self.enemy = enemy
            self.d = kwargs

        # получить экземпляр героя из партии, если такой есть
        # по умолчанию только первого попавшегося, при all=True - всех
        def hero(self, all=False):
            res = [ i for i in self.pers if i.kind == "hero" ]

            if all: return res

            if res: return res[0]

            return None

        # показать стрелки над головами
        def show_arrows(self):
            for i in self.pers: i.show_arrow()

        # скрыть стрелки над головами
        def hide_arrows(self):
            for i in self.pers: i.hide_arrow()

        # показать всю команду с сортировкой по расположению
        def show(self, repaint=False):
            xy = [ ]

            for i in b_heroes.pers + b_enemies.pers:
                i.x, i.y = i.get_pos()
                xy.append((i.x, i.y))

            xy.sort(key=lambda i: (i[1], i[0]))

            for i in self.pers:
                i.zorder = 1 + xy.index((i.x, i.y))

            for i in self.pers:
                i.show(repaint=repaint)

        # скрыть
        def hide(self, repaint=False):
            for i in self.pers: i.hide(repaint)

        # грохнуть перса
        def die(self, who):
            self.pers.remove(who)

            renpy.restart_interaction()

    # показать/скрыть стрелки над головами
    def b_arrows():
        if b_hero_turn():
            if b_state == "choose_action":
                b_enemies.hide_arrows()

                if b_who: b_who.show_arrow()

            elif b_state == "choose_target":
                b_heroes.hide_arrows()
                b_enemies.show_arrows()

            else:
                b_heroes.hide_arrows()
                b_enemies.hide_arrows()

    # показать всех в текущем состоянии
    def b_show_all(repaint=True):
        b_heroes.show()
        b_enemies.show()
        b_arrows()

        if repaint: renpy.restart_interaction()

    # сокращаем писанину
    def b_is_enemy(kind):
        # В боёвке герой/союзник — только hero/friend.
        # Все остальные типы, включая "enemy", "boss", "bandit" и т.п., считаются врагами.
        return kind not in ( "hero", "friend" )

    # клик по персонажу
    def b_clicked(who=None, alter=False):
        global b_who, b_target, b_state, b_current_index

        # Во время QTE клики обрабатывает modal-screen b_cancel_qte.
        # Нельзя делать Jump("end_cancel"), иначе первый бар выбрасывает нас из label b_cancel,
        # и серия из 2/3 QTE обрывается после первого бара.
        if b_cancel_on:
            return

        if who:
            # правая кнопка
            if alter:
                Show("b_info", who=who)()

            # ход игрока
            elif b_hero_turn():
                # выбор жертвы
                if b_state == "choose_target" and b_is_enemy(who.kind):
                    b_state = "choose_hit"
                    b_target = who
                    b_current_index = 0
                    b_arrows()
                    renpy.end_interaction(True)

                # информация о персонаже
                elif b_state != "action":
                    Show("b_info", who=who)()

    BClicked = renpy.curry(b_clicked)

    # наведение курсора на персонажа
    def b_hovered(who):
        global b_who_hover

        if who:
            b_who_hover = who

            if b_state == "choose_target":
                if b_hero_turn() and b_is_enemy(who.kind):
                    x, y = who.get_pos()
                    w, h = 300, who.h
                    renpy.show("b_corners", what=b_corners(w, h), at_list=[ b_corners_pos(x, y) ])

            renpy.restart_interaction()

    BHovered = renpy.curry(b_hovered)

    def b_unhovered():
        global b_who_hover

        b_who_hover = None
        renpy.hide("b_corners")
        renpy.restart_interaction()

    BUnhovered = renpy.curry(b_unhovered)

    # получить персонажа по его тегу
    def b_by_tag(tag):
        for i in list(b_heroes.pers + b_enemies.pers):
            if i.tag == tag: return i

        return None

    # список полей (тег, тип, текущая скорость) для сортировки
    def tag_kind_speed():
        return sorted(list([ (i.tag, i.kind, i.get_par("speed")) for i in list(b_heroes.pers + b_enemies.pers) ]), key=lambda i: i[2]) #, reverse=True)

    # порядок иконок вверху: текущий ход всегда первый слева, дальше очередь слева направо
    def tag_kind_speed_display():
        all = tag_kind_speed()

        if not all:
            return all

        index = max(0, min(b_index, len(all) - 1))
        return all[index:] + all[:index]

    # текущий персонаж
    def b_pers(index=None):
        if index is None: index = b_index

        return b_by_tag(tag_kind_speed()[index][0])

    # ход одного из персонажей игрока?
    def b_hero_turn():
        return not b_is_enemy(tag_kind_speed()[b_index][1])

    # первый живой персонаж игрока в очереди боя
    def b_first_hero_index():
        all = tag_kind_speed()

        for index, data in enumerate(all):
            tag, kind, speed = data

            if not b_is_enemy(kind):
                return index

        return 0

    # номер следующего в очереди на ход персонажа
    def b_next_index(index=None):
        if index is None:
            index = b_index

        index += 1

        if index >= len(b_heroes.pers + b_enemies.pers):
            index = 0

        return index

    # следующий в очереди на ход персонаж
    def b_next_pers():
        index = b_next_index()

        return b_pers(index)

    # есть ли сейчас незавершённая боевая анимация у любого персонажа
    def b_any_action_active():
        for i in list(b_heroes.pers + b_enemies.pers):
            if getattr(i, "act_i", None) is not None:
                return True

        return False

    # переключение последовательности анимаций во время атак
    def b_next_act(who):
        if who: who.next_act()

    BNextAct = renpy.curry(b_next_act)

    # применить отдельное попадание внутри составного удара
    def b_apply_attack_hit(who, act, p, hit_i=0, hit_count=1, token=None):
        if who:
            who.apply_attack_hit(act, p, hit_i, hit_count, token)

    BApplyAttackHit = renpy.curry(b_apply_attack_hit)


    # сколько QTE-баров нужно показать перед выбранной атакой
    def b_attack_bar_count(who, index, current=0):
        if who is None:
            return 1

        is_int = isinstance(index, int)
        hit_act = who.hit_acts[index] if is_int else b_other_acts[index]
        alias, name, ap, acts = hit_act

        if not acts or current >= len(acts):
            return 1

        act = acts[current][0]

        if act == "hit2":
            return 2

        if act == "hit3":
            return 3

        return 1

    # выбрать цель для ИИ
    def b_enemy_choose_target():
        alive = [i for i in b_heroes.pers if i.d["hp"] > 0]

        if alive:
            return alive[0]

        return None

    # выбрать удар для ИИ
    def b_enemy_choose_attack(who):
        if not who or not who.hit_acts:
            return "skip"

        allow = [i for i in range(len(who.hit_acts)) if who.hit_acts[i][2] <= who.ap]

        if not allow:
            return "skip"

        # Берём самый дорогой доступный удар, чтобы враг не тупил.
        allow.sort(key=lambda i: who.hit_acts[i][2], reverse=True)
        return allow[0]

    # это тренировочный манекен?
    def b_enemy_is_dummy(who):
        if not who:
            return False

        if who.spr_def == "dummy":
            return True

        return str(who.alias).lower() == str(_("Манекен")).lower()

    # ход манекена: он не атакует, а просто восстанавливает ровно +1 HP
    def b_enemy_dummy_turn(who):
        global b_dummy_heal_key, b_dummy_turn_done

        b_dummy_turn_done = True

        if not who:
            return

        # Защита от повторного срабатывания в рамках одного и того же хода.
        heal_key = "%s_%s" % (who.tag, b_turn_number)

        if b_dummy_heal_key == heal_key:
            return

        b_dummy_heal_key = heal_key

        # ВАЖНО: строго +1 к текущему HP. Не inc(), чтобы никакие сторонние эффекты
        # не дали визуально +2 на полоске.
        hp_old = int(who.d["hp"])
        hp_max = int(who.d["hp_max"])
        hp_new = min(hp_old + 1, hp_max)

        who.d["hp"] = hp_new
        who.show(repaint=True)

        if hp_new > hp_old:
            b_notify(_("+1 HP"), who, t=1.0)

    # переход хода к следующему персонажу
    def b_next_turn():
        global b_index, b_who, b_state, b_current_attack, b_current_index, b_target, b_who_hover, b_turn_number, b_dummy_turn_done

        b_current_attack = None
        b_current_index = None
        b_target = None
        b_who_hover = None
        b_dummy_turn_done = False

        b_index = b_next_index()
        b_turn_number += 1
        b_who = b_pers(b_index)

        if b_who:
            b_who.restore_ap()

        b_arrows()
        b_state = "choose_who"
        ### renpy.transition(dissolve)
        renpy.restart_interaction()

    # список id команд первого меню
    def b_button_items():
        return list([ i[1] for i in b_who.action_buttons() ])

    # команда №index из списка id команд первого меню
    def b_button_item(index):
        if isinstance(index, int):
            return b_button_items()[index]

        return index

    # выбран пропуск хода или другая заданная команда?
    def b_is_it(s="skip"):
        if isinstance(_return, int):
            return _return == b_button_items().index(s)

        return _return == s

    # взлетающий текст
    def b_notify(message, who=None, x=None, y=None, t=2):
        global b_dec_tag_index

        xx, yy = who.get_xy() if who else (.5, .5)

        if x is None: x = xx

        if y is None: y = yy

        tag = "b_notify_%s" % b_dec_tag_index
        b_dec_tag_index += 1
        id = "dec_%s" % b_dec_tag_index
        txt = Text(message, style="b_notify_text")
        renpy.show(id, what=txt, at_list=[ b_dec_at(x, y, t) ], zorder=111)
        b_dec_tag_index += 1
        if b_dec_tag_index > 11: b_dec_tag_index = 0
        renpy.restart_interaction()

    BNotify = renpy.curry(b_notify)

# стиль сообщения
style b_notify_text is b_button_text:
    color "#fff"

# взлетающий текст
# screen b_notify(message, x=.5, y=.5, t=2, style="b_notify_button"):
    # textbutton message:
        # style style
        # at b_dec_at(x, y, t)
        # action Hide()

    # if t > 0:
        # timer t repeat True action Hide()

define b_par = {
    "hp": _("Здоровье"),
    "strength": _("Сила"),
    "defense": _("Защита"),
    "speed": _("Скорость"),
    "hunger": _("Голод"),
    "moral": _("Мораль"),
    "lore": _("Знания")
}

define b_par_e = {
    "hp": _("Здоровье"),
    "strength": _("Сила"),
    "defense": _("Защита"),
    "speed": _("Скорость")
}

# информационное окно при клике по персонажу или противнику
screen b_info(who):
    modal True

    imagebutton idle "#1008" xysize(config.screen_width, config.screen_height) align(.5, .5) at b_show_hide action Hide()

    style_prefix "b"

    button:
        style "b_frame"
        at b_show_hide

        vbox:
            label who.alias
            null height 24

            $ par = b_par_e if b_is_enemy(who.kind) else b_par

            for i in par.keys():
                hbox:
                    frame style "empty" xsize 200:
                        text str(b_par[i]) color "#fdd"

                    text "%s {color=#ddf}(%s)" % (who.d[i], who.d[i + "_max"])

                null height 12

            null height 24
            textbutton _("OK") xsize 480 - 48 * 2 action Hide()

init python:
    # получить очки действия персонажа, если он задан
    def b_ap(who=None):
        # Если персонаж явно не передан — показываем ОД нашего героя,
        # а не текущего ходящего персонажа.
        who = who or b_heroes.hero() or b_who

        if who: return who.ap

        return 0

    # получить мордахен персонажа, если он задан
    # кэш портретов в верхнем интерфейсе,
    # чтобы анимация не перезапускалась при наведении мышкой
    b_face_cache = {}

    def b_face(who=None, xysize=(180, 180)):
        who = who or b_who

        if not who:
            return "null"

        # xysize может быть списком/кортежем, делаем нормальный ключ
        xysize_key = tuple(xysize)

        # Отдельный кэш для каждого персонажа, размера и базового спрайта.
        # Главное: displayable создаётся один раз, а не заново при hover.
        key = (who.tag, who.spr_def, who.spr.default_act, xysize_key)

        if key not in b_face_cache:
            face = who.sprite(who.spr.default_act, False)
            crop = pers_faces.get(who.spr_def, (0, 0, 1., 1.))
            face = Transform(face, crop=crop)

            b_face_cache[key] = AlphaMask(
                Transform(face, xysize=xysize),
                Transform("btn face mask", xysize=xysize)
            )

        return b_face_cache[key]

    # кто победил
    def b_get_end():
        if len(b_heroes.pers) <= 0: return "lose"

        if len(b_enemies.pers) <= 0: return "win"

        return False

    def b_is_choosen(who, tag):
        if who: return who.tag == tag

        return False

# навели курсор на персонажа/врага
default b_who_hover = None

# инфо по текущему персонажу вверху экрана
screen b_top():
    zorder 1

    key [ "skip", "toggle_skip", "fast_skip" ] action NullAction()

    timer .1 repeat True action SetField(config, "skipping", None)

    frame:
        style "empty"
        xysize (1920, 270)
        background "btn top"
        at transform:
            on show:
                alpha 0 yoffset -270
                ease .25 alpha 1 yoffset 0

            on hide:
                ease .25 alpha 0 yoffset -270

        hbox:
            ysize 235
            align(.5, 0)
            $ all = tag_kind_speed_display()

            for tag, kind, speed in all:
                $ xysize = (180, 180) if b_is_choosen(b_who, tag) else (110, 110)

                add "btn none" yalign .5

                if b_is_choosen(b_who, tag):
                    add "btn prev" yalign .5

                else:
                    add "btn none" yalign .5

                # очередь
                button:
                    style "empty"
                    xysize xysize
                    background ( "btn b180row" if xysize == (180, 180) else "btn b110row" )
                    hover_background ( "btn b180row hover" if xysize == (180, 180) else "btn b110row hover" )
                    align (.5, .5)
                    add b_face(b_by_tag(tag), xysize) align (.5, .5)
                    focus_mask True
                    hovered BHovered(b_by_tag(tag))
                    unhovered BUnhovered()
                    action BClicked(b_by_tag(tag))

            add "btn none" yalign .5

        # очки действий
        frame:
            style "empty"
            xysize (60, 60)
            background "btn b60"
            pos (173, 28)
            text str(b_ap()) color "#fff" align (.5, .5)

        # ромбики очков действий
        for i in range(b_ap()):
            add "btn b33" pos(233 + i * 16, 33 + ( 0 if i % 2 else 16 ))

        # кнопка выхода из боя
        # textbutton "{image=btn exit}" style "b_imagebutton" pos (70, 50) focus_mask True action SetVariable("b_end", "exit"), Jump("b_exit")

        # аватарка игрока слева сверху
        # ВАЖНО: это именно наша аватарка, она не меняется на манекена во время его хода.
        $ b_player_avatar = b_heroes.hero() or b_who

        button:
            style "empty"
            xysize (180, 180)
            background "btn b180"
            hover_background "btn b180 hover"
            pos (54, 32)
            add b_face(b_player_avatar) align (.5, .5)
            focus_mask True
            hovered BHovered(b_player_avatar)
            unhovered BUnhovered()
            action BClicked(b_player_avatar)

    # info
    if b_who_hover:
        frame:
            style "empty"
            background Frame("btn infobg", 0, 0)
            xysize (638, 60)
            align (.5, .93)
            at transform:
                on show:
                    alpha 0 yoffset 300
                    ease .25 alpha 1 yoffset 0

                on hide:
                    ease .25 alpha 0 yoffset 300

            text b_who_hover.alias align (.5, .5)

            #textbutton "{image=btn infoicon}" style "b_imagebutton" pos(50, 50) focus_mask True action BClicked(b_who_hover)

# кто атакует
default b_who = None

# кого атакуют
default b_target = None

# текущая анимация
default b_act = { }

# текущая часть составной анимации
default b_act_part = { }

# предыдущая анимация
default b_last_act_part = { }

# если в анимации требуется пауза
default b_pause = { }

# выбран пропуск хода
default b_skip = False

# "choose_action" - выбор действия (удар, спец-удар, артефакт, защита, пропуск хода)
# "choose_hit" - выбор типа удара
# "choose_combo" - выбор спец-удара
# "choose_item" - выбор предмета
# "choose_target" - выбор цели для атаки
# "timer" - время ожидания для уклонения
# "action" - выполняется какое-то действие
# "choose_who" - пауза для выбора текущего юнита
default b_state = "choose_who"

# список ударов по умолчанию для героев
# ( название, анимация, стоимость в ОД, коэффициент силы удара )
define b_hero_hit_acts_default = [
    ( _("Одиночный Удар"), "hit", 1, [ ("hit", .75) ] ),
    ( _("Двойной Удар"), "hit2", 2, [ ("hit2", .75) ] ),
    ( _("Тройной Удар"), "hit3", 3, [ ("hit3", .75) ] )
]

# список ударов по умолчанию для врагов
define b_enemy_hit_acts_default = [
    ( _("УДАР"), "hit", 1, [ ("hit", .55) ] )
]

# прочие действия
define b_other_acts = {
    "skip": ( _("ПРОПУСК ХОДА"), "skip", 0, [ ] ),
    "item": ( _("ПРЕДМЕТ"), "item", 0, [ ] )
}

# первое меню - выбор действия
define b_act_acts_default = [
    ( _("АТАКА"), "attack", 0, [ ] ),
    ( _("ЗАЩИТА"), "defense", 0, [ ] ),
    ( _("ПРЕДМЕТ"), "item", 0, [ ] ),
    ( _("ПРОПУСК ХОДА"), "skip", 0, [ ] )
]

# пустые команды
default b_heroes = Party(enemy=False)
default b_enemies = Party()

# пауза с условием
screen b_wait():
    layer "master"

    timer .05 repeat True action If(b_state == "choose_who" and not b_any_action_active(), Return())

# не реагировать на пробел и пкм
screen b_no_dismiss():
    layer "master"
    zorder 0

    button style "empty" xfill True yfill True action BClicked()

default b_end = False

# текущий персонаж
default b_index = 0

default b_current_index = None
default b_current_attack = None

# бой был вызван из книжного режима?
default b_return_to_book_after_battle = False

# какой режим был до боя
default b_book_mode_before_battle = None

# тест анимаций атаки
label battle(bg="bg battle room", allow_attack=None, allow_action=None):
    $ skipping_ = _skipping
    $ _skipping = None
    $ Preference("auto-forward", "disable")()
    hide screen quick_line
    show screen quick_line

    # Если бой вызван во время события в книге, книга должна временно исчезнуть.
    # Иначе экран book остаётся поверх master-слоя, а бой выглядит так,
    # будто он идёт "под книгой", не перед игроком.
    $ b_return_to_book_after_battle = mode in ( "book", "end" )
    $ b_book_mode_before_battle = mode

    if b_return_to_book_after_battle:
        $ mode = "battle"
        hide screen book
        hide screen choice
        window hide

    scene expression bg as bg
    show screen b_no_dismiss

    python:
        b_who_hover = None
        b_turn_number = 0
        b_dummy_heal_key = None
        b_dummy_turn_done = False
        b_index = b_first_hero_index()
        b_end = False
        b_show_all()

    with dissolve

    $ b_state = "choose_who"
    $ b_who = b_pers()

    show screen b_top

    # меняем в конце боя
    $ b_end = False

    # для выбора атаки или мульти-атаки
    $ b_current_index = None
    $ b_current_attack = None

    $ attack = False

# цикл боя
label battle_loop:
    if b_state == "choose_target":
        $ b_target = None

    if b_end != "exit":
        call screen b_wait()

    $ back = False

label battle_back:
    if b_end != "exit":
        python:
            if b_current_attack is None:
                b_skip = False
                b_dummy_turn_done = False
                _return = None
                x, y = b_who.get_pos()
                x -= 175
                y -= b_who.h // 2
                b_arrows()
                renpy.restart_interaction()

        # очки действий кончились, следующий персонаж
        if b_current_index is None:
            if b_who.ap < 1:
                $ b_next_turn()
                $ _return = "skip"

            # ОД ещё есть
            else:
                # выбор действия
                if b_is_enemy(b_who.kind):
                    if b_enemy_is_dummy(b_who):
                        # короткая пауза между окончанием хода игрока и началом хода соперника


                        # манекен не атакует — только лечится на +1 HP
                        $ b_enemy_dummy_turn(b_who)

                        # ждём, пока исчезнет всплывающая анимация +1 HP
                        $ renpy.pause(1.0, hard=True)

                        $ b_current_attack = None
                        $ b_current_index = None
                        $ _return = "skip"
                        $ attack = False

                    else:
                        $ b_target = b_enemy_choose_target()
                        $ b_current_attack = b_enemy_choose_attack(b_who)
                        $ b_current_index = 0
                        $ _return = "attack" if isinstance(b_current_attack, int) and b_target else "skip"
                        $ attack = True

                        if isinstance(b_current_attack, int):
                            $ alias, name, ap, acts = b_who.hit_acts[b_current_attack]
                            $ b_who.ap -= ap

                else:
                    $ rb_current = 0
                    $ rb_current_last = None
                    call screen rollbox(b_who.action_buttons(), x=x, y=y, allow=allow_action)
                    $ attack = False if _return else True

    $ b_skip = b_is_it()

    if b_end != "exit":
        # если выбран пропуск хода
        if b_skip:
            $ _return = "skip"

        elif b_is_it("attack") or not b_current_index is None:
            # выбор типа атаки
            $ x, y = renpy.get_mouse_pos()
            $ back = False

            if b_end != "exit":
                $ rb_current = 0
                $ rb_current_last = None

                if b_current_index is None and not b_is_enemy(b_who.kind):
                    call screen scrollbox(b_who.hit_buttons(), x=x, y=y, allow=allow_attack, back=True)
                    $ b_current_attack = _return
                    $ b_current_index = None

                    # отмена атаки
                    if _return >= len(b_who.hit_buttons()):
                        $ rb_current = 0
                        $ rb_current_last = None
                        $ b_current_attack = None
                        $ back = True

                        jump battle_back

                    # отнимаем очки действий
                    if isinstance(_return, int):
                        $ alias, name, ap, acts = b_who.hit_acts[_return]
                        $ b_who.ap -= ap

        elif b_is_it("defense"):
            # $ renpy.notify(_("В РАЗРАБОТКЕ..."))
            $ _return = "skip"

        elif b_is_it("item"):
            # $ renpy.notify(_("В РАЗРАБОТКЕ..."))
            $ _return = "skip"

        if _return != "skip" and b_end != "exit":
            if b_current_index is None:
                # выбираем цель
                if b_is_enemy(b_who.kind):
                    $ b_target = b_enemy_choose_target()
                    $ b_current_index = 0

                else:
                    # если есть из кого выбирать
                    if len(b_enemies.pers) > 1:
                        $ store.b_state = "choose_target"
                        $ b_arrows()
                        $ renpy.restart_interaction()

                        pause
                        hide b_corners

                    # если цель единственная
                    elif len(b_enemies.pers):
                        $ b_state = "choose_hit"
                        $ b_target = b_enemies.pers[0]
                        $ b_current_index = 0

            if b_end != "exit" and not b_current_index is None:
                # количество баров перед атакой: hit = 1, hit2 = 2, hit3 = 3
                $ b_bar_count = b_attack_bar_count(b_who, b_current_attack, b_current_index)
                $ b_cancel_results = [ ]

                # попытка усилить удар. Бары должны пройти ДО запуска анимации удара.
                if _return != "skip" and not b_is_enemy(b_who.kind) and b_end != "exit":
                    call b_cancel(b_bar_count) from _call_b_cancel

                # атакуем цель сразу после отработки бара/баров
                $ b_who.attack(b_current_attack, b_current_index)
                $ b_current_index += 1

                # кончилась ли череда атак
                if b_who.end_attacks(b_current_attack, b_current_index):
                    $ b_current_attack = None
                    $ b_current_index = None
                    $ b_skip = True

            pause

            if b_end != "exit":
                call screen b_wait()

        # костыль против ускорения
        if attack:
            $ renpy.pause(1.0, hard=True)

        # проверяем, кто помер
        python:
            for i in b_heroes.pers + b_enemies.pers:
                if i.d["hp"] <= 0: i.hide()

            die = False

            for i in b_heroes.pers + b_enemies.pers:
                if i.d["hp"] <= 0:
                    i.die()
                    die = True

                    if b_target and i.tag == b_target.tag:
                        b_current_attack = None
                        b_current_index = None

            if die:
                b_show_all(False)
                renpy.transition(dissolve)
                renpy.restart_interaction()
                pause(1)

        # очки действий кончились, следующий персонаж
        if b_end != "exit" and b_current_index is None:
            if (b_who.ap < 1) or b_skip:
                $ b_next_turn()

            $ b_end = b_get_end()

        # скрипт (навыки)
        # Манекену после его +1 HP не запускаем turn_exec(),
        # иначе внешние turn_end-скрипты могут дать ещё +1 и визуально получится +2.
        if b_who and not (b_dummy_turn_done and b_enemy_is_dummy(b_who)):
            $ b_who.turn_exec()

    if b_end != "exit" and not b_get_end():
        jump battle_loop

label b_exit:
    $ game_timers = { }
    $ b_heroes.hide()
    $ b_enemies.hide()
    hide screen b_wait
    hide screen b_top
    hide screen b_no_dismiss
    hide b_corners

    with dissolve

    hide screen quick_line
    show screen quick_line

    # Если бой запускался из книги, возвращаем книгу на экран,
    # чтобы событие спокойно продолжилось следующим текстом.
    if b_return_to_book_after_battle:
        $ mode = b_book_mode_before_battle
        window show
        show screen book

    $ _skipping = skipping_

    return

# успех qte-бара
default b_cancel = False

# сколько баров было успешно прожато перед текущей атакой
default b_cancel_hits = 0

# сколько всего баров было перед текущей атакой
default b_cancel_total = 1

# результаты каждого отдельного QTE перед текущей атакой
default b_cancel_results = [ ]

# qte-бар в процессе
default b_cancel_on = False

# время на qte-бар
define b_cancel_t = 1

transform b_align(x=.5, y=.5):
    align (x, y)

init python:
    # рандомное дробное число в заданных пределах
    # (f_to можно не указывать, тогда максимум берется из f_from)
    def rndf(f_from=0, f_to=None):
        if f_to is None:
            f_to = f_from
            f_from = .0
        return f_from + renpy.random.random() * (f_to - f_from)

# отдельный экран одного QTE-бара.
# ВАЖНО: это modal-screen, поэтому клик не проваливается в персонажей/кнопки боя.
screen b_cancel_qte(x=.5, y=.5, a1=270, a2=360, t=b_cancel_t):
    layer "master"
    zorder 1000
    modal True

    # ЛКМ = успешное нажатие QTE
    button:
        style "empty"
        xfill True
        yfill True
        action Return(True)

    add CircleBar(t=t, a1=a1, a2=a2) at b_align(x, y)

    # если не успели нажать — провал QTE
    timer t + .15 action Return(False)

    # ПРОБЕЛ или ЛКМ = успешное нажатие QTE
    key "K_SPACE" action Return(True)
    key "mousedown_1" action Return(True)


# попытка усилить удар через один или несколько qte-баров
label b_cancel(count=1):
    $ b_cancel_hits = 0
    $ b_cancel_total = max(1, int(count))
    $ b_cancel_left = b_cancel_total
    $ b_cancel_results = [ ]
    $ b_cancel = False

    while b_cancel_left > 0:
        $ b_cancel_on = True
        $ cb_value = 0
        $ a1, a2 = b_who.sector()
        $ b_cancel_x = rndf(.3, .7)
        $ b_cancel_y = rndf(.2, .6)

        call screen b_cancel_qte(b_cancel_x, b_cancel_y, a1, a2, b_cancel_t)

        $ b_cancel_on = False
        $ b_cancel_success = bool(_return) and (a1 <= cb_value * 360 <= a2)
        $ b_cancel_results.append(b_cancel_success)

        if b_cancel_success:
            $ b_cancel_hits += 1
            $ splay("cancel")
            $ b_notify(_("УСПЕХ!"), b_who)

        $ b_cancel_left -= 1

        if b_cancel_left > 0:
            pause .2

    $ b_cancel = b_cancel_hits > 0

    return
