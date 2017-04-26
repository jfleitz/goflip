/* 
 JAF 2017-04-16: Solenoid Driver controller. Now using USB on a Arduino Nano
2017-04-23: Some changes for performance:
1) Loop only method (not exiting from the Loop method, so calling SerialEvent from the Loop)
2) If a serial event is received to turn on a solenoid, do so asap
3) Changing the message format to 1 byte only
0bSSSS SDDD

SSSSS = Solenoid ID. 0-14 is a real solenoid. 15 is for direct controls.
DDD = Duration to keep the solenoid on:
000 = off
111 = on


Solenoid pos 15 will be used for controlling the direct lines for now.
0b0111 1 00 1


 { = flipper on = 7b = 0b0111 1011
 z = flipper off = 7a = 0b0111 1010


Test values:
10 
11
12
13
0b01010 100 = sol 10 = 0x54 = T
0x51 = Q = 0101 0 001
0b0101 1 001 = sol 11 = 0x59 = Y
0b0110 0 001 = sol 12 = 0x61 = a
ob0110 1 001 = sol 13 = 0x69 = i

18 - 1
0b1001 0 001 = x91



This will drive the SDU board using an Arduino Nano.
To communicate, the NAK is set to "c". 
Message is in the following format:

{[SolID][Duration/Cmd]}

Duration/Cmd is as followed:
0 = off (allows to force the off)
1 = pulse (default time, whatever a default pulse is (50ms?))
2-254 (02-0xfe) = 10ms*the value passed in (so up to 2.54 sec)

SolID 16 = on board LED


If SolID is > 16, then it is considered a "Direct Control" as followed:

SolID   |  Control
-----------------------------
17      |  Turns on Relay1
18      |  Turns on Relay 2
19      |  Turns on J2/10
20      |  Turns on J2/9

 


Send a NAK when ACK is receieved ("|") 

0000 0 000
1000 1 111

 
 
 P2 
 1 = Relay 1 (active high)
 2 = Relay 2 (active high)
 3 = J2/10  (active high)
 4 = J2/9 (active high)
 5 = Addr 1
 6 = Addr 2
 7 = Key
 8 = Addr 3
 9 = Addr 4

Nano:
Low PortD (Active High)
 4=sol Address 0 
 5=sol Address 1
 6=sol Address 2
 7=sol Address 3
 
 Low PortB (Active High as well):
 0=str Relay 1
 1=str Relay 2
 2=str J2/10
 3=str J2/9
 */

#include <avr/io.h>
//#define DEBUG true
#define F_CPU 16000000UL  //16mhz clock
#include <util/delay.h> 
#define SDU_NAK 0x63 //  c

// Pin 13 has an LED connected on most Arduino boards.
// PB5 is the LED
// give it a name:
int led = 13; 

volatile boolean decrement;

//holds the status of the lamp.
//00=off, 01=on, 10=slow blink, 11=fast blink
volatile unsigned char _solControl[17]; 


enum recvstate {
  SolID = 1,
  ControlID,
  EOL
};


boolean ack = false;

// the setup routine runs once when you press reset:
void setup() {                
  // initialize the digital pin as an output.
  pinMode(led, OUTPUT);     
  Serial.begin(57600); // start serial for output. Should only have this if debugging

  decrement = false;
  SetSolAll(0);

  DDRD = DDRD | B11110000; //Port D4-D7 output for Lamp0-Lamp3
  DDRB = DDRB | B00001111; //Port B0-B3 output for Lamp Strobe0-Strobe3

  // initialize timer1 
  noInterrupts();           // disable all interrupts
  TCCR1A = 0;
  TCCR1B = 0;
  TCNT1  = 0;

  OCR1A =  3125;           // compare match register 16MHz/256/20Hz 
  TCCR1B |= (1 << WGM12);   // CTC mode
  TCCR1B |= (1 << CS12);    // 256 prescaler 
  TIMSK1 |= (1 << OCIE1A);  // enable timer compare interrupt
  interrupts();             // enable all interrupts

  #ifdef DEBUG
  Serial.println("==SDU initialized==");
  #endif
  
  ack=true;
  
 // Serial.println("Initialized!");
}

void SetSolAll(int value){
  int i;

  for(i=0;i<=16;i++){
    _solControl[i]=value;
  }
}

void SetSolenoid(int sol, unsigned char data){
  //0=off, 1=on, 2=slow blink, 3=fast blink, 4=brightness
  _solControl[sol]=data;
}

// the loop routine runs over and over again forever:
void loop() {
  int i = 0;
  boolean dodecrement=false;
  boolean nothingOn = true;
  
  while(1){
    if (serialEventRun) serialEventRun(); //doing this, since we are staying in the loop
    
    if(ack){
      Serial.write(SDU_NAK);
      ack=false;
    }
     
    if(_solControl[i] >0) {      
      if(i<16){
       turnOnSol(i);
       nothingOn = false;
      }
      
      //step it down if we are ready (unless set to always on which is 'z')
      if(dodecrement && _solControl[i]<7){
        _solControl[i]--;
      }
       
    }
 
   //prepping for the next complete loop:
    i++;
    if(i>16){
      i=0;
      if (nothingOn) turnOffSols();  //we don't want to hold the last Solenoid on forever, so turn off if nothing was turned on.
      nothingOn=true; //reset the bit.
          //just loop through each lamp value...
      if(decrement){
        decrement=false;
        dodecrement=true;
      } else {
        dodecrement = false ;
      }
    }
  }
}



ISR(TIMER1_COMPA_vect)          // timer compare interrupt service routine
{  
  decrement=true;
  /*for(i=0;i<=16;i++){
    if(_solControl[i] >0 && _solControl[i]<74){
        _solControl[i]--;
    }
  }*/
}


void serialEvent() {
  static  byte solCommand;
  static byte solID;
/*  
0xSSSS SDDD

SSSSS = Solenoid ID. 0-15 is a real solenoid. 16+ is for direct controls.
DDD = Duration to keep the solenoid on:
000 = off
111 = on
*/
  
  while (Serial.available()) {
    //_receiveState
     
    char inChar = Serial.read(); 

    switch(inChar){
      case '|': //7c = 0x0111 1100  
        ack = true;
        break;
      default:
           solID = inChar>>3;
           solCommand = inChar & 0x07;
           
          #ifdef DEBUG
            Serial.print("solID Set to:");
            Serial.println(solID,DEC);
            
            Serial.print("Value passed: ");
            Serial.println(solCommand, DEC);
            
          #endif
           
          if(solID < 15){
              _solControl[solID] = solCommand;
              if(solCommand>0){
                turnOnSol(solID); //act now
              }
          }else{           
            directSols(solCommand>>1 , solCommand & 0x01); 
          }
    }    
  }
}


/*
 J1 
Low PortD (Active High)
 4=sol Address 0 
 5=sol Address 1
 6=sol Address 2
 7=sol Address 3
*/

void turnOnSol(int solID){
  unsigned char newValue;

  newValue = PORTD | 0xf0;  //reset high nibble to all 1111####
  newValue = newValue & (solID<<4 | 0x0f); //example for 0xE0, this would be 0x0f AND-ing with. so new value would be 0000####
  PORTD = newValue;
}

void turnOnLED(){
  //          Serial.println ("Turning On LED");
  //          PORTB ^=(1<<PB5);
  PORTB = PORTB | 0x20; //turn on PB5 = LED 
  //          _delay_ms(500); 

}
void turnOffLED(){
  PORTB = PORTB & 0xdf; //turn off PB5 = LED   
}

void turnOffSols(){
  PORTD |= 0xf0; //turn "off" the high bits.   
}

void turnOffAll(){
  PORTD |= 0xf0; //turn "off" the high bits.   
  PORTB &= 0xf0; //turn off the low bits
}

/*
 Low PortB (Active High as well):
 0=str Relay 1
 1=str Relay 2
 2=str J2/10
 3=str J2/9

SolID   |  Control
-----------------------------
17      |  Turns on Relay1
18      |  Turns on Relay 2
19      |  Turns on J2/10
20      |  Turns on J2/9
*/
void directSols(byte solID, byte cmd){
  #ifdef DEBUG
    Serial.print("directSols called for:");
    Serial.println(solID,DEC);
    Serial.println(cmd,DEC);
    Serial.println("----");
    {
    z
    
  #endif
  if(cmd>0){
    switch(solID){
      case 00:  //Relay1
        PORTB|=0x01;
        break;
      case 01:  //Relay2
         PORTB|=0x02;
         break; 
      case 10:  //J2/10
        PORTB|=0x04;
        break;
      case 11:  //J2/9
        PORTB|=0x08;
        break;
    }
  }else{
    switch(solID){
      case 00:  //Relay1
        PORTB&=0xfe;
        break;
      case 01:  //Relay2
        PORTB&=0xfd;
        break; 
      case 10:  //J2/10
        PORTB&=0xfb;
        break;
      case 11:  //J2/9
        PORTB&=0xf7;
        break;
    }    
  }
}
