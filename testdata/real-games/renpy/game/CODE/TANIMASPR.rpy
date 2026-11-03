## КЛАССЫ СПРАЙТОВ

# селена
define fselena = TAnimaSpr("selena")

# чучело
define fdummy = TAnimaSpr("dummy", h=280)

## ДАЛЕЕ ЛУЧШЕ НИЧЕГО НЕ МЕНЯТЬ

# долгая пауза для loop == False
define YEAR = 365.25 * 86400

init -99 python:
    # список папок, в которых лежат спрайты спрайты, внутри которых все их анимации
    anima_tags = [ "fspr" ]

    # время на один кадр анимации (.05 - минимальное)
    anima_t = .05

    # словарь исключений
    anima_t_not = { "stay": .1 }

    # переход между кадрами, например: Dissolve(.1)
    anima_trans = None

    # исключения (None - между кадрами нет переходов)
    # anima_trans_not = { "move": Dissolve(.1) }
    anima_trans_not = { }

    # по умолчанию анимации не зацикленные
    anima_loop = False

    # исключения
    anima_loop_not = { "walk": True, "stay": True }

    # рост персонажа по умолчанию (противника)
    anima_char_h = 280

    # размеры спрайтов
    anima_xysize = (667, 500)

    # якорь для разных типов юнитов
    anima_xanchor = 175
    anima_xanchor_enemy = anima_xysize[0] - 175
    # исключения (например, чтобы удар пришёлся точно по врагу, а не рядом)
    anima_xanchors = {
        "hit": anima_xysize[0] - 200,
        "hit2": anima_xysize[0] - 200,
        "hit3": anima_xysize[0] - 200
    }

    # звуки на конкретных кадрах составных hit-анимаций
    # hit2: звук на 1-м кадре первого и второго блока 1-5
    # hit3: звук на 1-м кадре каждого из трёх блоков 1-5
    anima_frame_sounds = {
        "hit2": { 0: "b_hit", 5: "b_hit" },
        "hit3": { 0: "b_hit", 5: "b_hit", 10: "b_hit" }
    }

    ## ЗДЕСЬ НЕ МЕНЯТЬ
    # словарь со списками кадров для анимаций
    anima_dict = { }

# трансформы для спрайтов
transform a_xzoom(xz=1):
    xzoom xz

transform a_anchor(xa=.5, ya=1.):
    anchor (xa, ya)

transform a_pos(x=.5, y=.5):
    pos (x, y)

init -2 python:
    # класс для анимации, основанный на TransitionAnimation,
    # но с возможностью перезапуска анимации в любой момент,
    class TAnima(renpy.display.core.Displayable):
        def __init__(self, *args, **properties):
            properties.setdefault('style', 'animation')
            self.anim_timebase = properties.pop('anim_timebase', True)
            self.frame_sounds = properties.pop('frame_sounds', { })
            self._last_sound_frame = None
            super(TAnima, self).__init__(**properties)
            images = [ ]
            delays = [ ]
            transitions = [ ]

            for i, arg in enumerate(args):
                if i % 3 == 0:
                    images.append(renpy.easy.displayable(arg))

                elif i % 3 == 1:
                    delays.append(arg)

                else:
                    transitions.append(arg)

            if len(images) > len(delays): delays.append(YEAR)

            if len(images) > len(transitions): transitions.append(None)

            self.images = images
            self.prev_images = [ images[-1] ] + images[:-1]
            self.delays = delays
            self.transitions = [ transitions[-1] ] + transitions[:-1]

            self.t = 0
            self.st = None

            # по умолчанию стартует без паузы
            self.pause_on_start = properties.get("pause", False)
            self.paused = self.pause_on_start

        # None - переключить состояние паузы
        # True/False - включить/отключить паузу
        def setpause(self, *args, on=None, **kwarg):
            self.paused = not self.paused if on is None else on

        # поставить на паузу
        def pause(self, *args, **kwarg):
            self.setpause(True)

        # снять с паузы
        def unpause(self, *args, **kwarg):
            self.setpause(False)

        # перезапуск анимации
        def restart(self, *args, **kwarg):
            self.t = 0
            self.st = None
            self.paused = self.pause_on_start
            self._last_sound_frame = None

        # путаю reset и restart, пусть будут обе)
        def reset(self, *args, **kwarg):
            self.restart(*args, **kwarg)

        def render(self, width, height, st, at):
            tt = at if self.anim_timebase else st

            if self.st is None: self.st = tt

            if not self.paused: self.t += tt - self.st

            self.st = tt
            self.t %= sum(self.delays)
            t = self.t
            delays = self.delays[:]

            if delays[-1] == YEAR: delays[-1] = 0

            for frame_index, (image, prev, delay, trans) in enumerate(zip(self.images, self.prev_images, delays, self.transitions)):
                if t < delay or delay == 0:
                    if not renpy.game.less_updates:
                        renpy.display.render.redraw(self, delay - t)

                    if frame_index != self._last_sound_frame:
                        self._last_sound_frame = frame_index
                        sound_name = self.frame_sounds.get(frame_index, None)

                        if sound_name:
                            splay(sound_name)

                    if delays[-1] < 365.25 * 86400.0:
                        if trans and self.t >= self.delays[0]:
                            image = trans(old_widget=prev, new_widget=image)

                    im = renpy.display.render.render(image, width, height, t, at)
                    width, height = im.get_size()
                    rv = renpy.display.render.Render(width, height)
                    rv.blit(im, (0, 0))

                    return rv

                else:
                    t = t - delay

        def visit(self):
            return self.images

    # упрощённое автоматическое создание анимации
    # пример эффекта effect=Dissolve(.1)
    # (по умолчанию зациклено)
    def Anima(tag, count, t=.1, effect=None, loop=True, two_digits=False, **properties):
        args = [ ]
        properties["loop"] = loop

        for i in range(1, count + 1):
            s = "%02d" % (i) if two_digits else str(i)
            args.append(tag + s)

            if loop or (i < count):
                args.append(t)
                args.append(effect)

        return TAnima(*args, **properties)

    # класс для спрайта с контролируемыми анимациями
    class TAnimaSpr(object):
        def __init__(self, tag, default_act="stay", enemy=False, h=anima_char_h):
            super(TAnimaSpr, self).__init__()
            self.tag = tag
            self.act = default_act[:]
            self.default_act = default_act[:]
            self.spr = None
            self.enemy = enemy
            self.h = h
            # self.at_list = at_list or [ ]
            self.set_act()

        # перезапустить анимацию
        def reset(self):
            if self.spr: self.spr.reset()

        # перезапустить анимацию
        def restart(self):
            if self.spr: self.spr.restart()

        # переключить паузу анимации
        def setpause(self, on=None):
            if self.spr: self.spr.setpause(on)

        # поставить на паузу анимацию
        def pause(self):
            if self.spr: self.spr.pause()

        # снять с паузы анимацию
        def unpause(self):
            if self.spr: self.spr.unpause()

        # получить список кадров для обычной или составной hit-анимации
        # hit2: первые 5 кадров hit повторяются 2 раза, потом идут остальные кадры
        # hit3: первые 5 кадров hit повторяются 3 раза, потом идут остальные кадры
        def get_act_images(self, act):
            k = self.tag + " " + act

            if k in anima_dict.keys():
                return anima_dict[k]

            if act in ("hit2", "hit3"):
                base_k = self.tag + " hit"

                if base_k in anima_dict.keys():
                    images = anima_dict[base_k]
                    repeat_count = 2 if act == "hit2" else 3
                    first = images[:5]
                    rest = images[5:]

                    if first:
                        return first * repeat_count + rest

            return None

        # сменить анимацию
        def set_act(self, act=None, restart=True):
            if not isinstance(act, (int, float)):
                act = act or self.act[:] or self.default_act[:]
                k = self.tag + " " + act

                images = self.get_act_images(act)

                # если существует
                if images:
                    # время на кадр анимации (учитываем исключения)
                    t = anima_t_not.get(act, anima_t)
                    trans = anima_trans_not.get(act, anima_trans)
                    args = sum([ [ i, t, trans ] for i in images ], [ ])

                    if len(args) > 2 and not anima_loop_not.get(act, anima_loop): args = args[:-2]

                    self.spr = TAnima(*args, frame_sounds=anima_frame_sounds.get(act, { }))

                self.act = act[:]

                # при необходимости перезапускаем анимацию
                if restart and self.spr: self.restart()

        # длительность анимации
        def get_act_t(self, act=None):
            if isinstance(act, (int, float)): return act

            t = 0
            act = act or self.act[:] or self.default_act[:]
            k = self.tag + " " + act

            images = self.get_act_images(act)

            # если существует
            if images:
                # время на кадр анимации (учитываем исключения) * количество кадров
                t = anima_t_not.get(act, anima_t) * len(images)

            # не меньше .1, иначе экран не успеет обновиться
            return t

        # якорь слева для союзников, справа - для врагов, по центру - для спецприёмов
        def xanchor(self, act=None):
            xanchor = anima_xanchor_enemy if self.enemy else anima_xanchor

            return anima_xanchors.get(act or self.act, xanchor)

        # якорь, отзеркаливание и положение спрайта
        def get_at_list(self, x=None, y=None, act=None, nobar=True):
            return [ a_anchor(self.xanchor(act)), a_xzoom(-1 if self.enemy and nobar else 1), a_pos(x, y) ]

        # получить текущий спрайт
        def sprite(self, act=None, restart=None, **kwargs):
            # если анимация меняется, то перезапускаем её
            if (restart is None) and (act != self.act): restart = True

            self.set_act(act, restart)
            hp = kwargs.pop("hp", None)

            return Transform(self.spr, **kwargs)

        # хелсбар, если он нужен
        def hpbar(self, act=None, **kwargs):
            act = act or self.act
            hp = kwargs.pop("hp", None)
            x, y = kwargs.pop("x", None), kwargs.pop("y", None)

            # если в параметрах есть hp [0..1], показать полоску над головой
            if b_stay(act) and not hp is None and not x is None and not y is None:
                y -= self.h - b_bar_xysize[1] - 1

                # return x, y, b_bar(hp, z=kwargs.get("z", 0))
                return x, y, b_bar(hp)

            return None

        # убрать с экрана
        def hide(self, tag, repaint=False):
            renpy.hide(tag)
            renpy.hide("%s_hp" % tag)
            renpy.hide("%s_alias" % tag)
            renpy.hide("%s_alias2" % tag)
            renpy.hide("%s_arrow" % tag)

            if repaint: renpy.restart_interaction()

        # показать стрелку над головой
        def show_arrow(self, x, y, tag=None, zorder=0):
            a = "%s_arrow" % (tag or self.tag)
            y -= self.h - b_bar_xysize[1] - 1
            renpy.show(a, what="arrow", at_list=[ a_arrow_at(x, y, 50) ], zorder=zorder)

        # скрыть стрелку над головой
        def hide_arrow(self, tag=None):
            renpy.hide("%s_arrow" % (tag or self.tag))

        # показать спрайт на экране
        def show(self, act=None, tag=None, restart=None, repaint=True, at_list=None, zorder=0, who=None, **kwargs):
            act = act or self.act
            hpbar = self.hpbar(act=act, **kwargs)
            x, y = kwargs.pop("x", None), kwargs.pop("y", None)
            at_list0 = self.get_at_list(x, y, act)
            at_list = at_list or [ ]
            at_list = at_list + at_list0
            spr = self.sprite(act, restart, **kwargs)
            kwargs["z"] = zorder
            t = tag or self.tag

            spr = ImageButton(spr, focus_mask=True, clicked=BClicked(who), hovered=BHovered(who), unhovered=BUnhovered(), alternate=BClicked(who, True))
            at_list = [ b_hover_at ] + at_list # + make_list(self.at_list)

            renpy.show(t, what=spr, at_list=at_list, zorder=zorder)

            b = "%s_hp" % t
            n = "%s_alias" % t
            n2 = "%s_alias2" % t

            at_list0 = [ ]

            if hpbar:
                x, y, bar = hpbar
                renpy.show(b, what=bar, at_list=[ a_bar_at(x, y, 5) ], zorder=zorder)

                if who.alias:
                    renpy.show(n, what=Text(str(who.alias), style="b_alias"), at_list=[ a_bar_at(x, y, 35) ], zorder=zorder)
                    renpy.show(n2, what=Text("{b}HP{/b}:%s" % int(who.get_par("hp")), style="b_alias"), at_list=[ a_bar_at(x, y, 15) ], zorder=zorder)

            else:
                renpy.hide(b)
                renpy.hide(n)
                renpy.hide(n2)

            if repaint: renpy.restart_interaction()

# сдвиг для хелсбара
transform a_bar_at(x=0, y=0, yo=0, t=.25):
    parallel:
        anchor (.5, 1.)
        pos (x, y - yo)

    parallel:
        on show:
            alpha 0 xoffset 0
            linear t alpha 1

        on hide:
            linear t alpha 0 xoffset 111

# стрелочка
transform a_arrow_at(x=0, y=0, alias=0, t=.2, dy=10, ty=1):
    parallel:
        subpixel True
        anchor (.5, 1.)
        xpos x
        ypos y - alias
        ease ty * .5 ypos y - alias - dy
        ease ty * .5 ypos y - alias
        repeat

    parallel:
        on show:
            alpha 0
            linear t alpha .75

        on hide:
            linear t alpha 0

init 2000 python:
    import re

    # натуральная сортировка (1, 2, 10, 20...)
    def natsorted(l):
        convert = lambda text: float(text) if text.isdigit() else text
        alphanum = lambda key: [ convert(c) for c in re.split('([-+]?[0-9]*\.?[0-9]*)', key) ]
        l.sort(key=alphanum)

        return l

    # создать списки данных для движений по именам из списка тегов
    # { "dane walk": [ "dane walk 1", "dane walk 2", ... ]
    def anima_auto():
        global anima_dict

        # все объявленные изображения
        for img in renpy.list_images():
            # разбиваем на теги
            tags = img.split()

            # добавляем очередной кадр анимации
            if len(tags) > 3:
                if tags[0] in anima_tags:
                    k = tags[1] + " " + tags[2]

                    if not k in anima_dict.keys(): anima_dict[k] = [ ]

                    anima_dict[k].append(img)
                    anima_dict[k] = natsorted(anima_dict[k])

    anima_auto()