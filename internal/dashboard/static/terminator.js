/*!
 * Leaflet.Terminator — Overlay day/night region on a Leaflet map
 * Copyright (c) 2013 Joerg Dietrich <astro@joergdietrich.com>
 * MIT License — see licenses/LEAFLET-TERMINATOR-MIT-LICENSE
 */
(function (global, factory) {
	typeof exports === 'object' && typeof module !== 'undefined' ? module.exports = factory(require('leaflet')) :
	typeof define === 'function' && define.amd ? define(['leaflet'], factory) :
	(global = typeof globalThis !== 'undefined' ? globalThis : global || self, (global.L = global.L || {}, global.L.terminator = factory(global.L)));
})(this, (function (L) { 'use strict';

	function julian(date) {
		return (date / 86400000) + 2440587.5;
	}

	function GMST(julianDay) {
		var d = julianDay - 2451545.0;
		return (18.697374558 + 24.06570982441908 * d) % 24;
	}

	var Terminator = L.Polygon.extend({
		options: {
			interactive: false,
			color: '#00',
			opacity: 0.5,
			fillColor: '#00',
			fillOpacity: 0.5,
			resolution: 2,
			longitudeRange: 720
		},

		initialize: function (options) {
			this.version = '0.1.0';
			this._R2D = 180 / Math.PI;
			this._D2R = Math.PI / 180;
			L.Util.setOptions(this, options);
			var latLng = this._compute(this.options.time);
			this.setLatLngs(latLng);
		},

		setTime: function (date) {
			this.options.time = date;
			var latLng = this._compute(date);
			this.setLatLngs(latLng);
		},

		_sunEclipticPosition: function (julianDay) {
			var n = julianDay - 2451545.0;
			var L = 280.460 + 0.9856474 * n;
			L %= 360;
			var g = 357.528 + 0.9856003 * n;
			g %= 360;
			var lambda = L + 1.915 * Math.sin(g * this._D2R) +
				0.02 * Math.sin(2 * g * this._D2R);
			var R = 1.00014 - 0.01671 * Math.cos(g * this._D2R) -
				0.0014 * Math.cos(2 * g * this._D2R);
			return {lambda: lambda, R: R};
		},

		_eclipticObliquity: function (julianDay) {
			var n = julianDay - 2451545.0;
			var T = n / 36525;
			var epsilon = 23.43929111 -
				T * (46.836769 / 3600
					- T * (0.0001831 / 3600
						+ T * (0.00200340 / 3600
							- T * (0.576e-6 / 3600
								- T * 4.34e-8 / 3600))));
			return epsilon;
		},

		_sunEquatorialPosition: function (sunEclLng, eclObliq) {
			var alpha = Math.atan(Math.cos(eclObliq * this._D2R)
				* Math.tan(sunEclLng * this._D2R)) * this._R2D;
			var delta = Math.asin(Math.sin(eclObliq * this._D2R)
				* Math.sin(sunEclLng * this._D2R)) * this._R2D;

			var lQuadrant = Math.floor(sunEclLng / 90) * 90;
			var raQuadrant = Math.floor(alpha / 90) * 90;
			alpha = alpha + (lQuadrant - raQuadrant);

			return {alpha: alpha, delta: delta};
		},

		_hourAngle: function (lng, sunPos, gst) {
			var lst = gst + lng / 15;
			return lst * 15 - sunPos.alpha;
		},

		_latitude: function (ha, sunPos) {
			var lat = Math.atan(-Math.cos(ha * this._D2R) /
				Math.tan(sunPos.delta * this._D2R)) * this._R2D;
			return lat;
		},

		_compute: function (time) {
			var today = time ? new Date(time) : new Date();
			var julianDay = julian(today);
			var gst = GMST(julianDay);
			var latLng = [];

			var sunEclPos = this._sunEclipticPosition(julianDay);
			var eclObliq = this._eclipticObliquity(julianDay);
			var sunEqPos = this._sunEquatorialPosition(sunEclPos.lambda, eclObliq);
			for (var i = 0; i <= this.options.longitudeRange * this.options.resolution; i++) {
				var lng = -this.options.longitudeRange/2 + i / this.options.resolution;
				var ha = this._hourAngle(lng, sunEqPos, gst);
				latLng[i + 1] = [this._latitude(ha, sunEqPos), lng];
			}
			if (sunEqPos.delta < 0) {
				latLng[0] = [90, -this.options.longitudeRange/2];
				latLng[latLng.length] = [90, this.options.longitudeRange/2];
			} else {
				latLng[0] = [-90, -this.options.longitudeRange/2];
				latLng[latLng.length] = [-90, this.options.longitudeRange/2];
			}
			return latLng;
		}
	});

	function terminator(options) {
		return new Terminator(options);
	}

	return terminator;

}));
